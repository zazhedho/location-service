package importer

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"

	domainlocation "location-service/internal/domain/location"
	domainpopulation "location-service/internal/domain/population"
)

type PopulationImportStats struct {
	RowsRead         int
	RowsParsed       int
	RowsImported     int
	RowsSkipped      int
	NationalRows     int
	UnknownCodes     int
	UnsupportedCodes int
}

var (
	populationInsertPattern = regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+wilayah_penduduk\b`)
	populationCodePattern   = regexp.MustCompile(`^[0-9]{2}(?:\.[0-9]{2})?$`)
)

func ImportPopulation(ctx context.Context, db *sql.DB, path string) (PopulationImportStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return PopulationImportStats{}, err
	}
	defer file.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return PopulationImportStats{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		CREATE TEMP TABLE population_import_staging (
			code varchar(13) PRIMARY KEY,
			male bigint NOT NULL,
			female bigint NOT NULL,
			total bigint NOT NULL,
			source varchar(255) NOT NULL,
			reference_date date NOT NULL
		) ON COMMIT DROP`); err != nil {
		return PopulationImportStats{}, err
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"population_import_staging",
		"code", "male", "female", "total", "source", "reference_date",
	))
	if err != nil {
		return PopulationImportStats{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = stmt.Close()
		}
	}()

	result := PopulationImportStats{}
	parsed, err := ParsePopulationTuples(file, func(item domainpopulation.Population) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.Code == "0" {
			result.NationalRows++
			result.RowsSkipped++
			return nil
		}
		if !populationCodePattern.MatchString(item.Code) {
			result.UnsupportedCodes++
			result.RowsSkipped++
			return nil
		}
		_, err := stmt.ExecContext(ctx, item.Code, item.Male, item.Female, item.Total, item.Source, item.ReferenceDate)
		return err
	})
	if err != nil {
		return PopulationImportStats{}, err
	}
	result.RowsRead = parsed.RowsRead
	result.RowsParsed = parsed.RowsParsed

	if _, err := stmt.ExecContext(ctx); err != nil {
		return PopulationImportStats{}, fmt.Errorf("flush population copy: %w", err)
	}
	if err := stmt.Close(); err != nil {
		return PopulationImportStats{}, fmt.Errorf("close population copy: %w", err)
	}
	closed = true

	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM population_import_staging s
		LEFT JOIN raw_locations l ON l.code = s.code
		WHERE l.code IS NULL`).Scan(&result.UnknownCodes); err != nil {
		return PopulationImportStats{}, err
	}
	result.RowsSkipped += result.UnknownCodes

	inserted, err := tx.ExecContext(ctx, `
		INSERT INTO location_population (code, male, female, total, source, reference_date)
		SELECT s.code, s.male, s.female, s.total, s.source, s.reference_date
		FROM population_import_staging s
		JOIN raw_locations l ON l.code = s.code
		WHERE l.level IN (1, 2)
		ON CONFLICT (code) DO UPDATE SET
			male = EXCLUDED.male,
			female = EXCLUDED.female,
			total = EXCLUDED.total,
			source = EXCLUDED.source,
			reference_date = EXCLUDED.reference_date,
			imported_at = now()`)
	if err != nil {
		return PopulationImportStats{}, fmt.Errorf("store population: %w", err)
	}
	affected, err := inserted.RowsAffected()
	if err != nil {
		return PopulationImportStats{}, err
	}
	result.RowsImported = int(affected)

	if err := tx.Commit(); err != nil {
		return PopulationImportStats{}, err
	}
	return result, nil
}

func ParsePopulationTuples(r io.Reader, emit func(domainpopulation.Population) error) (PopulationImportStats, error) {
	if emit == nil {
		return PopulationImportStats{}, errors.New("population tuple callback is required")
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var stats PopulationImportStats
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
			item, err := parsePopulationTuple(tuple)
			if err != nil {
				return fmt.Errorf("population row %d: %w", stats.RowsRead, err)
			}
			stats.RowsParsed++
			if err := emit(item); err != nil {
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
			if !populationInsertPattern.MatchString(line) {
				continue
			}
			inInsert = true
			if match := valuesPattern.FindStringIndex(line); match != nil {
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
		return stats, errors.New("unterminated wilayah_penduduk INSERT")
	}
	return stats, nil
}

func parsePopulationTuple(raw string) (domainpopulation.Population, error) {
	fields, err := splitSQLFields(raw)
	if err != nil {
		return domainpopulation.Population{}, err
	}
	if len(fields) != 5 {
		return domainpopulation.Population{}, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}

	code, present, err := textValue(fields[0])
	if err != nil || !present {
		return domainpopulation.Population{}, errors.New("population code is required")
	}
	code = strings.TrimSpace(code)
	if code != "0" && !domainlocation.IsValidCode(code) {
		return domainpopulation.Population{}, errors.New("invalid population code")
	}

	name, present, err := textValue(fields[1])
	if err != nil || !present || strings.TrimSpace(name) == "" {
		return domainpopulation.Population{}, errors.New("population name is required")
	}
	male, err := populationCount(fields[2], "male")
	if err != nil {
		return domainpopulation.Population{}, err
	}
	female, err := populationCount(fields[3], "female")
	if err != nil {
		return domainpopulation.Population{}, err
	}
	total, err := populationCount(fields[4], "total")
	if err != nil {
		return domainpopulation.Population{}, err
	}
	if male > int64(1<<63-1)-female || male+female != total {
		return domainpopulation.Population{}, errors.New("male plus female must equal total")
	}

	return domainpopulation.Population{
		Code:          code,
		Name:          name,
		Male:          male,
		Female:        female,
		Total:         total,
		Source:        domainpopulation.Source,
		ReferenceDate: domainpopulation.ReferenceDate,
	}, nil
}

func populationCount(raw, name string) (int64, error) {
	value, present, err := textValue(raw)
	if err != nil || !present {
		return 0, fmt.Errorf("%s must be a nonnegative integer", name)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s must be a nonnegative integer", name)
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("%s must be a nonnegative integer", name)
		}
	}
	number, err := strconv.ParseUint(value, 10, 63)
	if err != nil {
		return 0, fmt.Errorf("%s must be a nonnegative integer", name)
	}
	return int64(number), nil
}
