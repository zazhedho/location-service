package importer

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/lib/pq"

	domainarea "location-service/internal/domain/area"
	domainlocation "location-service/internal/domain/location"
)

type AreaImportStats struct {
	RowsRead           int
	RowsImported       int
	RowsSkippedUnknown int
	CodeCorrections    int
	NameCorrections    int
}

type parsedArea struct {
	item          domainarea.Area
	codeCorrected bool
	nameCorrected bool
}

var (
	areaInsertPattern   = regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+wilayah_luas\b`)
	areaValuesPattern   = regexp.MustCompile(`(?i)\bVALUES\b`)
	areaCodeCorrections = map[string]string{
		"11.1": "11.10",
		"12.1": "12.10",
		"12.2": "12.20",
		"14.1": "14.10",
		"31.0": "31",
	}
	areaNameCorrections = map[string]map[string]string{
		"53.02": {"Kab Timor Tengah Selatan": "Kabupaten Timor Tengah Selatan"},
		"96":    {"Papua Barat Oaya": "Papua Barat Daya"},
	}
)

func ImportAreas(ctx context.Context, db *sql.DB, path string) (AreaImportStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return AreaImportStats{}, err
	}
	defer file.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return AreaImportStats{}, err
	}
	defer tx.Rollback()

	knownCodes, err := rawCodeSet(ctx, tx)
	if err != nil {
		return AreaImportStats{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE TEMP TABLE location_area_import_staging (
			code varchar(13) PRIMARY KEY,
			name varchar(100) NOT NULL,
			area_km2 double precision NOT NULL,
			CHECK (
				area_km2 >= 0
				AND area_km2 <> 'NaN'::double precision
				AND area_km2 <> 'Infinity'::double precision
				AND area_km2 <> '-Infinity'::double precision
			)
		) ON COMMIT DROP`); err != nil {
		return AreaImportStats{}, err
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("location_area_import_staging", "code", "name", "area_km2"))
	if err != nil {
		return AreaImportStats{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = stmt.Close()
		}
	}()

	unknownCodes := 0
	stats, err := ParseAreaTuples(file, func(item domainarea.Area) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok := knownCodes[item.Code]; !ok {
			unknownCodes++
			return nil
		}
		if _, err := stmt.ExecContext(ctx, item.Code, item.Name, item.AreaKM2); err != nil {
			return fmt.Errorf("stage area %s: %w", item.Code, err)
		}
		return nil
	})
	if err != nil {
		return AreaImportStats{}, fmt.Errorf("parse area file %s: %w", path, err)
	}
	stats.RowsSkippedUnknown = unknownCodes

	if _, err := stmt.ExecContext(ctx); err != nil {
		return AreaImportStats{}, fmt.Errorf("flush area copy: %w", err)
	}
	if err := stmt.Close(); err != nil {
		return AreaImportStats{}, fmt.Errorf("close area copy: %w", err)
	}
	closed = true

	result, err := tx.ExecContext(ctx, `
		INSERT INTO location_areas (code, area_km2, source, reference_date, imported_at)
		SELECT s.code, s.area_km2, $1, $2::date, now()
		FROM location_area_import_staging s
		JOIN raw_locations l ON l.code = s.code
		ON CONFLICT (code) DO UPDATE SET
			area_km2 = EXCLUDED.area_km2,
			source = EXCLUDED.source,
			reference_date = EXCLUDED.reference_date,
			imported_at = EXCLUDED.imported_at`,
		domainarea.Source, domainarea.ReferenceDate)
	if err != nil {
		return AreaImportStats{}, fmt.Errorf("store areas: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return AreaImportStats{}, fmt.Errorf("count imported areas: %w", err)
	}
	expected := stats.RowsRead - stats.RowsSkippedUnknown
	if int(affected) != expected {
		return AreaImportStats{}, fmt.Errorf("store areas: matched %d rows, want %d", affected, expected)
	}
	stats.RowsImported = int(affected)

	if err := tx.Commit(); err != nil {
		return AreaImportStats{}, err
	}
	return stats, nil
}

func ParseAreaTuples(r io.Reader, emit func(domainarea.Area) error) (AreaImportStats, error) {
	if emit == nil {
		return AreaImportStats{}, errors.New("area tuple callback is required")
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var stats AreaImportStats
	var pending string
	inInsert := false
	lineNumber := 0

	process := func(input string) error {
		pending += input
		tuples, remainder, done, err := consumeTuples(pending)
		if err != nil {
			return err
		}
		pending = remainder
		for _, tuple := range tuples {
			stats.RowsRead++
			parsed, err := parseAreaTuple(tuple)
			if err != nil {
				return fmt.Errorf("area tuple %d: %w", stats.RowsRead, err)
			}
			if parsed.codeCorrected {
				stats.CodeCorrections++
			}
			if parsed.nameCorrected {
				stats.NameCorrections++
			}
			if err := emit(parsed.item); err != nil {
				return err
			}
		}
		if done {
			inInsert = false
			pending = ""
		}
		return nil
	}

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if !inInsert {
			if !areaInsertPattern.MatchString(line) {
				continue
			}
			inInsert = true
			if match := areaValuesPattern.FindStringIndex(line); match != nil {
				if err := process(line[match[1]:]); err != nil {
					return stats, fmt.Errorf("line %d: %w", lineNumber, err)
				}
			}
			continue
		}
		if err := process(line + "\n"); err != nil {
			return stats, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return stats, err
	}
	if inInsert {
		return stats, errors.New("unterminated wilayah_luas INSERT")
	}
	return stats, nil
}

func parseAreaTuple(raw string) (parsedArea, error) {
	fields, err := splitSQLFields(raw)
	if err != nil {
		return parsedArea{}, err
	}
	if len(fields) != 3 {
		return parsedArea{}, fmt.Errorf("expected 3 fields, got %d", len(fields))
	}

	rawCode, present, err := textValue(fields[0])
	if err != nil || !present {
		return parsedArea{}, errors.New("invalid area code")
	}
	code, codeCorrected, err := normalizeAreaCode(rawCode)
	if err != nil {
		return parsedArea{}, err
	}

	rawName, present, err := textValue(fields[1])
	if err != nil || !present || strings.TrimSpace(rawName) == "" {
		return parsedArea{}, errors.New("area name is required")
	}
	name := strings.TrimSpace(rawName)
	nameCorrected := false
	if corrected, ok := areaNameCorrections[code][name]; ok {
		name = corrected
		nameCorrected = true
	}
	area, err := parseAreaValue(fields[2])
	if err != nil {
		return parsedArea{}, err
	}

	return parsedArea{
		item: domainarea.Area{
			Code:          code,
			Name:          name,
			AreaKM2:       area,
			Source:        domainarea.Source,
			ReferenceDate: domainarea.ReferenceDate,
		},
		codeCorrected: codeCorrected,
		nameCorrected: nameCorrected,
	}, nil
}

func parseAreaValue(raw string) (float64, error) {
	area, err := floatValueFromSQL(raw)
	if err != nil || area == nil || math.IsNaN(*area) || math.IsInf(*area, 0) || *area < 0 {
		return 0, errors.New("invalid area_km2")
	}
	return *area, nil
}

func normalizeAreaCode(raw string) (string, bool, error) {
	code := strings.TrimSpace(raw)
	corrected, changed := areaCodeCorrections[code]
	if changed {
		code = corrected
	}
	if !domainlocation.IsValidCode(code) {
		return "", false, fmt.Errorf("invalid area code %q", raw)
	}
	if strings.Count(code, ".") > 1 {
		return "", false, errors.New("area code must be a province or regency/city code")
	}
	return code, changed, nil
}
