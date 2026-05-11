package importer

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/lib/pq"

	domainlocation "location-service/internal/domain/location"
)

var tuplePattern = regexp.MustCompile(`\('((?:''|[^'])*)','((?:''|[^'])*)'\)`)

func Import(ctx context.Context, db *sql.DB, path string, truncate bool) (domainlocation.ImportStats, error) {
	if !truncate {
		return domainlocation.ImportStats{}, errors.New("bulk import requires truncate=true")
	}
	file, err := os.Open(path)
	if err != nil {
		return domainlocation.ImportStats{}, err
	}
	defer file.Close()

	rawRows := make([][]any, 0, 100000)
	provinceRows := make([][]any, 0, 40)
	regencyRows := make([][]any, 0, 600)
	districtRows := make([][]any, 0, 8000)
	villageRows := make([][]any, 0, 90000)

	stats := domainlocation.ImportStats{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		matches := tuplePattern.FindAllStringSubmatch(scanner.Text(), -1)
		for _, match := range matches {
			code := strings.TrimSpace(unescapeSQL(match[1]))
			name := strings.TrimSpace(unescapeSQL(match[2]))
			if code == "" || name == "" {
				continue
			}
			parts := strings.Split(code, ".")
			level := len(parts)
			if level < 1 || level > 4 {
				continue
			}
			rawRows = append(rawRows, []any{code, name, level})
			stats.Raw++
			switch level {
			case 1:
				provinceRows = append(provinceRows, []any{code, name, code})
				stats.Provinces++
			case 2:
				regencyCode := parts[0] + "." + parts[1]
				regencyRows = append(regencyRows, []any{regencyCode, parts[1], parts[0], name, code})
				stats.Regencies++
			case 3:
				regencyCode := parts[0] + "." + parts[1]
				districtCode := regencyCode + "." + parts[2]
				districtRows = append(districtRows, []any{districtCode, parts[2], parts[0], regencyCode, name, code})
				stats.Districts++
			case 4:
				regencyCode := parts[0] + "." + parts[1]
				districtCode := regencyCode + "." + parts[2]
				villageRows = append(villageRows, []any{code, parts[3], parts[0], regencyCode, districtCode, name, code})
				stats.Villages++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return domainlocation.ImportStats{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domainlocation.ImportStats{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `TRUNCATE villages, districts, regencies, provinces, raw_locations RESTART IDENTITY`); err != nil {
		return domainlocation.ImportStats{}, err
	}
	if err := copyRows(ctx, tx, "raw_locations", []string{"code", "name", "level"}, rawRows); err != nil {
		return domainlocation.ImportStats{}, err
	}
	if err := copyRows(ctx, tx, "provinces", []string{"code", "name", "source_code"}, provinceRows); err != nil {
		return domainlocation.ImportStats{}, err
	}
	if err := copyRows(ctx, tx, "regencies", []string{"code", "short_code", "province_code", "name", "source_code"}, regencyRows); err != nil {
		return domainlocation.ImportStats{}, err
	}
	if err := copyRows(ctx, tx, "districts", []string{"code", "short_code", "province_code", "regency_code", "name", "source_code"}, districtRows); err != nil {
		return domainlocation.ImportStats{}, err
	}
	if err := copyRows(ctx, tx, "villages", []string{"code", "short_code", "province_code", "regency_code", "district_code", "name", "source_code"}, villageRows); err != nil {
		return domainlocation.ImportStats{}, err
	}
	if err := tx.Commit(); err != nil {
		return domainlocation.ImportStats{}, err
	}
	return stats, nil
}

func copyRows(ctx context.Context, tx *sql.Tx, table string, columns []string, rows [][]any) error {
	if len(rows) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(table, columns...))
	if err != nil {
		return fmt.Errorf("prepare copy %s: %w", table, err)
	}
	defer stmt.Close()
	for _, row := range rows {
		if _, err := stmt.ExecContext(ctx, row...); err != nil {
			return fmt.Errorf("copy %s: %w", table, err)
		}
	}
	if _, err := stmt.ExecContext(ctx); err != nil {
		return fmt.Errorf("flush copy %s: %w", table, err)
	}
	return nil
}

func unescapeSQL(value string) string {
	return strings.ReplaceAll(value, "''", "'")
}
