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
	"strings"
)

var (
	postalInsertPattern   = regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+wilayah_kodepos\b`)
	postalValuesPattern   = regexp.MustCompile(`(?i)\bVALUES\b`)
	postalLocationPattern = regexp.MustCompile(`^[0-9]{2}\.[0-9]{2}\.[0-9]{2}\.[0-9]{4}$`)
	postalValuePattern    = regexp.MustCompile(`^[0-9]{5}$`)
)

// ResolvePostalCodeFile returns the configured postal-code seed when it exists.
// When POSTAL_CODE_FILE is unset, data/kodepos.sql is used.
func ResolvePostalCodeFile() (string, bool, error) {
	path := strings.TrimSpace(os.Getenv("POSTAL_CODE_FILE"))
	if path == "" {
		path = "data/kodepos.sql"
	}
	if _, err := os.Stat(path); err == nil {
		return path, true, nil
	} else if !os.IsNotExist(err) {
		return path, false, err
	}
	return path, false, nil
}

func LoadPostalCodes(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	postalCodes := make(map[string]string)
	_, err = ParsePostalTuples(file, func(code, postalCode string) error {
		if previous, exists := postalCodes[code]; exists && previous != postalCode {
			return fmt.Errorf("conflicting postal codes for %s", code)
		}
		postalCodes[code] = postalCode
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse postal-code file %s: %w", path, err)
	}
	if len(postalCodes) == 0 {
		return nil, fmt.Errorf("postal-code file %s contains no rows", path)
	}
	return postalCodes, nil
}

func ParsePostalTuples(r io.Reader, emit func(code, postalCode string) error) (int, error) {
	if emit == nil {
		return 0, errors.New("postal tuple callback is required")
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	rows := 0
	pending := ""
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
			rows++
			code, postalCode, err := parsePostalTuple(tuple)
			if err != nil {
				return fmt.Errorf("postal row %d: %w", rows, err)
			}
			if err := emit(code, postalCode); err != nil {
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
			if !postalInsertPattern.MatchString(line) {
				continue
			}
			inInsert = true
			if match := postalValuesPattern.FindStringIndex(line); match != nil {
				if err := process(line[match[1]:]); err != nil {
					return rows, fmt.Errorf("line %d: %w", lineNumber, err)
				}
			}
			continue
		}
		if err := process(line + "\n"); err != nil {
			return rows, fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return rows, err
	}
	if inInsert {
		return rows, errors.New("unterminated wilayah_kodepos INSERT")
	}
	return rows, nil
}

func parsePostalTuple(raw string) (string, string, error) {
	fields, err := splitSQLFields(raw)
	if err != nil {
		return "", "", err
	}
	if len(fields) != 2 {
		return "", "", fmt.Errorf("expected 2 fields, got %d", len(fields))
	}
	code, present, err := textValue(fields[0])
	if err != nil || !present || !postalLocationPattern.MatchString(code) {
		return "", "", errors.New("invalid postal location code")
	}
	postalCode, present, err := textValue(fields[1])
	if err != nil || !present || !postalValuePattern.MatchString(postalCode) {
		return "", "", errors.New("invalid postal code")
	}
	return code, postalCode, nil
}

func ImportPostalCodes(ctx context.Context, db *sql.DB, path string) (int, error) {
	postalCodes, err := LoadPostalCodes(path)
	if err != nil {
		return 0, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		CREATE TEMP TABLE location_postal_import_staging (
			code varchar(13) PRIMARY KEY,
			postal_code varchar(5) NOT NULL CHECK (postal_code ~ '^[0-9]{5}$')
		) ON COMMIT DROP`); err != nil {
		return 0, err
	}

	rows := make([][]any, 0, len(postalCodes))
	for code, postalCode := range postalCodes {
		rows = append(rows, []any{code, postalCode})
	}
	if err := copyRows(ctx, tx, "location_postal_import_staging", []string{"code", "postal_code"}, rows); err != nil {
		return 0, err
	}

	var unknown int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM location_postal_import_staging s
		LEFT JOIN villages v ON v.code = s.code
		WHERE v.code IS NULL`).Scan(&unknown); err != nil {
		return 0, err
	}
	if unknown > 0 {
		return 0, fmt.Errorf("postal-code rows do not match %d villages", unknown)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE villages v
		SET postal_code = s.postal_code
		FROM location_postal_import_staging s
		WHERE v.code = s.code`)
	if err != nil {
		return 0, fmt.Errorf("store postal codes: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if int(affected) != len(postalCodes) {
		return 0, fmt.Errorf("updated %d villages, want %d", affected, len(postalCodes))
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(affected), nil
}
