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
	"strconv"
	"strings"

	domainisland "location-service/internal/domain/island"
)

type IslandImportStats struct {
	RowsRead       int
	RowsImported   int
	DuplicateCodes int
	RowsSkipped    int
}

var (
	islandInsertPattern = regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+wilayah_pulau\b`)
	valuesPattern       = regexp.MustCompile(`(?i)\bVALUES\b`)
	islandCodePattern   = regexp.MustCompile(`^[0-9]{2}\.[0-9]{2}\.[0-9]{5}$`)
)

func ImportIslands(ctx context.Context, db *sql.DB, path string) (IslandImportStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return IslandImportStats{}, err
	}
	defer file.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return IslandImportStats{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `TRUNCATE islands`); err != nil {
		return IslandImportStats{}, err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO islands (code, province_code, name, latitude, longitude, status, area, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		return IslandImportStats{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = stmt.Close()
		}
	}()

	var counts IslandImportStats
	stats, err := ParseIslandTuples(file, func(item domainisland.Island) error {
		result, err := stmt.ExecContext(ctx,
			item.Code,
			item.ProvinceCode,
			item.Name,
			floatValue(item.Latitude),
			floatValue(item.Longitude),
			item.Status,
			floatValue(item.Area),
			item.Notes,
		)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			counts.DuplicateCodes++
		} else {
			counts.RowsImported++
		}
		return err
	})
	if err != nil {
		return IslandImportStats{}, err
	}
	if err := stmt.Close(); err != nil {
		return IslandImportStats{}, err
	}
	closed = true
	stats.RowsImported = counts.RowsImported
	stats.DuplicateCodes = counts.DuplicateCodes
	if err := tx.Commit(); err != nil {
		return IslandImportStats{}, err
	}
	committed = true
	return stats, nil
}

func ParseIslandTuples(r io.Reader, emit func(domainisland.Island) error) (IslandImportStats, error) {
	if emit == nil {
		return IslandImportStats{}, errors.New("island tuple callback is required")
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var stats IslandImportStats
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
			item, err := parseIslandTuple(tuple)
			if err != nil {
				stats.RowsSkipped++
				continue
			}
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
			if !islandInsertPattern.MatchString(line) {
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
		return stats, errors.New("unterminated wilayah_pulau INSERT")
	}
	return stats, nil
}

func consumeTuples(input string) (tuples []string, remainder string, done bool, err error) {
	open := -1
	inQuote := false
	for i := 0; i < len(input); i++ {
		char := input[i]
		if open < 0 {
			switch char {
			case '(':
				open = i
			case ';':
				return tuples, "", true, nil
			}
			continue
		}

		if inQuote {
			if char == '\'' {
				if i+1 < len(input) && input[i+1] == '\'' {
					i++
				} else {
					inQuote = false
				}
			}
			continue
		}
		switch char {
		case '\'':
			inQuote = true
		case ')':
			tuples = append(tuples, input[open+1:i])
			open = -1
		}
	}
	if open >= 0 {
		return tuples, input[open:], false, nil
	}
	return tuples, "", false, nil
}

func parseIslandTuple(raw string) (domainisland.Island, error) {
	fields, err := splitSQLFields(raw)
	if err != nil {
		return domainisland.Island{}, err
	}
	if len(fields) != 5 && len(fields) != 7 {
		return domainisland.Island{}, fmt.Errorf("expected 5 or 7 fields, got %d", len(fields))
	}

	code, present, err := textValue(fields[0])
	if err != nil || !present || !islandCodePattern.MatchString(code) {
		return domainisland.Island{}, errors.New("invalid island code")
	}
	name, present, err := textValue(fields[1])
	if err != nil || !present || strings.TrimSpace(name) == "" {
		return domainisland.Island{}, errors.New("island name is required")
	}
	latitude, err := floatValueFromSQL(fields[2])
	if err != nil || latitude != nil && (*latitude < -90 || *latitude > 90) {
		return domainisland.Island{}, errors.New("invalid island latitude")
	}
	longitude, err := floatValueFromSQL(fields[3])
	if err != nil || longitude != nil && (*longitude < -180 || *longitude > 180) {
		return domainisland.Island{}, errors.New("invalid island longitude")
	}
	status, _, err := textValue(fields[4])
	if err != nil {
		return domainisland.Island{}, errors.New("invalid island status")
	}

	var area *float64
	var notes string
	if len(fields) == 7 {
		notes, _, err = textValue(fields[6])
		if err != nil {
			return domainisland.Island{}, errors.New("invalid island notes")
		}
		if latitude == nil && longitude == nil && (notes == "BP" || notes == "TBP") {
			area, err = floatValueFromSQL(fields[4])
			if err == nil && area != nil {
				// Source has one row shifted right: area, notes, then status.
				status = notes
				notes, _, err = textValue(fields[5])
			}
		} else {
			area, err = floatValueFromSQL(fields[5])
			if err != nil && notes == "" {
				// Source has one known row with an unquoted note shifted into the area column.
				notes, _, err = textValue(fields[5])
				area = nil
			}
		}
		if err != nil || area != nil && (math.IsNaN(*area) || math.IsInf(*area, 0) || *area < 0) {
			return domainisland.Island{}, errors.New("invalid island area")
		}
		if len(status) > 10 && notes == "" {
			// Source uses the status column for notes on some rows.
			notes, status = status, ""
		}
	}
	if len(status) > 10 {
		return domainisland.Island{}, errors.New("invalid island status")
	}

	return domainisland.Island{
		Code:         code,
		ProvinceCode: code[:2],
		Name:         name,
		Latitude:     latitude,
		Longitude:    longitude,
		Status:       status,
		Area:         area,
		Notes:        notes,
	}, nil
}

func splitSQLFields(raw string) ([]string, error) {
	fields := make([]string, 0, 7)
	start := 0
	inQuote := false
	for i := 0; i < len(raw); i++ {
		char := raw[i]
		if inQuote {
			if char == '\'' {
				if i+1 < len(raw) && raw[i+1] == '\'' {
					i++
				} else {
					inQuote = false
				}
			}
			continue
		}
		if char == '\'' {
			inQuote = true
		} else if char == ',' {
			fields = append(fields, strings.TrimSpace(raw[start:i]))
			start = i + 1
		}
	}
	if inQuote {
		return nil, errors.New("unterminated quoted field")
	}
	fields = append(fields, strings.TrimSpace(raw[start:]))
	return fields, nil
}

func textValue(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if strings.EqualFold(raw, "NULL") {
		return "", false, nil
	}
	if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
		return strings.ReplaceAll(raw[1:len(raw)-1], "''", "'"), true, nil
	}
	if strings.Contains(raw, "'") {
		return "", false, errors.New("invalid quoted field")
	}
	return raw, true, nil
}

func floatValueFromSQL(raw string) (*float64, error) {
	value, present, err := textValue(raw)
	if err != nil || !present || value == "" {
		return nil, err
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return nil, errors.New("invalid number")
	}
	return &number, nil
}

func floatValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
