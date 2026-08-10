package importer

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"

	"location-service/internal/boundary"
	domainlocation "location-service/internal/domain/location"
	"location-service/pkg/storage"
)

type BoundaryImportStats struct {
	Read           int
	Imported       int
	SkippedUnknown int
}

type boundaryRow struct {
	Code        string
	Coordinates domainlocation.Coordinates
	LeafletPath json.RawMessage
}

const (
	boundaryUploadWorkers = 8
	boundaryUploadTimeout = 60 * time.Second
)

func BoundaryFiles(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*-boundaries-*.sql.gz"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no boundary gzip files found in %s", dir)
	}
	return paths, nil
}

func ImportBoundaries(ctx context.Context, db *sql.DB, provider storage.Provider, paths []string) (BoundaryImportStats, error) {
	if len(paths) == 0 {
		return BoundaryImportStats{}, errors.New("at least one boundary gzip file is required")
	}
	if provider == nil {
		return BoundaryImportStats{}, errors.New("storage is required for boundary import")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return BoundaryImportStats{}, err
	}
	defer tx.Rollback()

	knownCodes, err := rawCodeSet(ctx, tx)
	if err != nil {
		return BoundaryImportStats{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE TEMP TABLE boundary_import_staging (
			code varchar(13) PRIMARY KEY,
			centroid_lat double precision NOT NULL,
			centroid_lng double precision NOT NULL,
			object_key text NOT NULL
		) ON COMMIT DROP`); err != nil {
		return BoundaryImportStats{}, err
	}

	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("boundary_import_staging", "code", "centroid_lat", "centroid_lng", "object_key"))
	if err != nil {
		return BoundaryImportStats{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = stmt.Close()
		}
	}()

	stats := BoundaryImportStats{}
	existingKeys, err := existingBoundaryKeys(ctx, provider)
	if err != nil {
		return BoundaryImportStats{}, fmt.Errorf("list existing boundary objects: %w", err)
	}
	for _, path := range paths {
		if err := uploadBoundaryFile(ctx, provider, path, knownCodes, existingKeys); err != nil {
			return BoundaryImportStats{}, err
		}

		file, err := os.Open(path)
		if err != nil {
			return BoundaryImportStats{}, err
		}
		gzipFile, err := gzip.NewReader(file)
		if err != nil {
			_ = file.Close()
			return BoundaryImportStats{}, fmt.Errorf("open gzip %s: %w", path, err)
		}

		parseErr := parseBoundarySQL(gzipFile, func(row boundaryRow) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			stats.Read++
			if _, ok := knownCodes[row.Code]; !ok {
				stats.SkippedUnknown++
				return nil
			}
			key := boundaryObjectKey(row.Code)
			if _, err := stmt.ExecContext(ctx, row.Code, row.Coordinates.Latitude, row.Coordinates.Longitude, key); err != nil {
				return err
			}
			stats.Imported++
			return nil
		})
		gzipErr := gzipFile.Close()
		fileErr := file.Close()
		if parseErr != nil {
			return BoundaryImportStats{}, fmt.Errorf("parse boundary file %s: %w", path, parseErr)
		}
		if gzipErr != nil {
			return BoundaryImportStats{}, fmt.Errorf("close gzip %s: %w", path, gzipErr)
		}
		if fileErr != nil {
			return BoundaryImportStats{}, fmt.Errorf("close boundary file %s: %w", path, fileErr)
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		return BoundaryImportStats{}, fmt.Errorf("flush boundary copy: %w", err)
	}
	if err := stmt.Close(); err != nil {
		return BoundaryImportStats{}, fmt.Errorf("close boundary copy: %w", err)
	}
	closed = true

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO location_boundaries (code, centroid_lat, centroid_lng, object_key, leaflet_path)
		SELECT code, centroid_lat, centroid_lng, object_key, NULL
		FROM boundary_import_staging
		ON CONFLICT (code) DO UPDATE SET
			centroid_lat = EXCLUDED.centroid_lat,
			centroid_lng = EXCLUDED.centroid_lng,
			object_key = EXCLUDED.object_key,
			leaflet_path = NULL,
			imported_at = now()`); err != nil {
		return BoundaryImportStats{}, fmt.Errorf("store boundaries: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return BoundaryImportStats{}, err
	}
	return stats, nil
}

func uploadBoundaryFile(ctx context.Context, provider storage.Provider, path string, knownCodes, existingKeys map[string]struct{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	gzipFile, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("open gzip %s: %w", path, err)
	}

	pool := newBoundaryUploadPool(ctx, provider)
	parseErr := parseBoundarySQL(gzipFile, func(row boundaryRow) error {
		if _, ok := knownCodes[row.Code]; !ok {
			return nil
		}
		if _, ok := existingKeys[boundaryObjectKey(row.Code)]; ok {
			return nil
		}
		return pool.Submit(row)
	})
	gzipErr := gzipFile.Close()
	fileErr := file.Close()
	uploadErr := pool.Close()
	if uploadErr != nil {
		return fmt.Errorf("upload boundary file %s: %w", path, uploadErr)
	}
	if parseErr != nil {
		return fmt.Errorf("parse boundary file %s: %w", path, parseErr)
	}
	if gzipErr != nil {
		return fmt.Errorf("close gzip %s: %w", path, gzipErr)
	}
	if fileErr != nil {
		return fmt.Errorf("close boundary file %s: %w", path, fileErr)
	}
	return nil
}

type boundaryUploadPool struct {
	ctx      context.Context
	cancel   context.CancelFunc
	provider storage.Provider
	jobs     chan boundaryRow
	wg       sync.WaitGroup
	mu       sync.Mutex
	err      error
}

func newBoundaryUploadPool(ctx context.Context, provider storage.Provider) *boundaryUploadPool {
	poolContext, cancel := context.WithCancel(ctx)
	pool := &boundaryUploadPool{
		ctx:      poolContext,
		cancel:   cancel,
		provider: provider,
		jobs:     make(chan boundaryRow, boundaryUploadWorkers),
	}
	for worker := 0; worker < boundaryUploadWorkers; worker++ {
		pool.wg.Add(1)
		go pool.run()
	}
	return pool
}

func (p *boundaryUploadPool) run() {
	defer p.wg.Done()
	for row := range p.jobs {
		if p.ctx.Err() != nil {
			return
		}
		uploadCtx, cancel := context.WithTimeout(p.ctx, boundaryUploadTimeout)
		err := uploadBoundaryObject(uploadCtx, p.provider, row)
		cancel()
		if err != nil {
			p.fail(err)
			return
		}
	}
}

func (p *boundaryUploadPool) Submit(row boundaryRow) error {
	select {
	case p.jobs <- row:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

func (p *boundaryUploadPool) fail(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	if p.err == nil {
		p.err = err
	}
	p.mu.Unlock()
}

func (p *boundaryUploadPool) Close() error {
	close(p.jobs)
	p.wg.Wait()
	p.cancel()
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func boundaryObjectKey(code string) string {
	return "boundaries/" + code + ".json.gz"
}

func uploadBoundaryObject(ctx context.Context, provider storage.Provider, row boundaryRow) error {
	payload, err := boundary.EncodeBoundaryPayload(row.LeafletPath)
	if err != nil {
		return err
	}
	key := boundaryObjectKey(row.Code)
	if err := provider.Upload(ctx, key, bytes.NewReader(payload), int64(len(payload)), "application/gzip"); err != nil {
		return fmt.Errorf("upload boundary %s: %w", row.Code, err)
	}
	return nil
}

func existingBoundaryKeys(ctx context.Context, provider storage.Provider) (map[string]struct{}, error) {
	lister, ok := provider.(storage.PrefixLister)
	if !ok {
		return map[string]struct{}{}, nil
	}
	return lister.List(ctx, "boundaries/")
}

func rawCodeSet(ctx context.Context, tx *sql.Tx) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `SELECT code FROM raw_locations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	codes := make(map[string]struct{})
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes[code] = struct{}{}
	}
	return codes, rows.Err()
}

func parseBoundarySQL(input io.Reader, emit func(boundaryRow) error) error {
	reader := bufio.NewReaderSize(input, 64*1024)
	var statement strings.Builder
	var token strings.Builder
	inString := false
	inValues := false
	rowNumber := 0

	for {
		value, err := reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if inValues {
			if value == '(' {
				rowNumber++
				row, err := readBoundaryTuple(reader)
				if err != nil {
					return fmt.Errorf("boundary row %d: %w", rowNumber, err)
				}
				if err := emit(row); err != nil {
					return fmt.Errorf("boundary row %d: %w", rowNumber, err)
				}
				continue
			}
			if value == ';' {
				inValues = false
				statement.Reset()
				token.Reset()
			}
			continue
		}

		if inString {
			if value == '\'' {
				next, readErr := reader.ReadByte()
				if readErr == nil && next == '\'' {
					continue
				}
				if readErr == nil {
					if err := reader.UnreadByte(); err != nil {
						return err
					}
				}
				inString = false
			}
			continue
		}

		if value == '\'' {
			inString = true
			statement.WriteByte(value)
			continue
		}
		statement.WriteByte(value)
		if statement.Len() > 4096 {
			trimmed := statement.String()
			statement.Reset()
			statement.WriteString(trimmed[len(trimmed)-4096:])
		}
		if isSQLIdentifierByte(value) {
			token.WriteByte(toLowerASCII(value))
			continue
		}
		if token.String() == "values" && strings.Contains(strings.ToLower(statement.String()), "insert into wilayah_boundaries") {
			inValues = true
		}
		token.Reset()
		if value == ';' {
			statement.Reset()
		}
	}
}

func readBoundaryTuple(reader *bufio.Reader) (boundaryRow, error) {
	fields := make([]string, 5)
	for index := range fields {
		quoted := index == 0 || index == 1 || index == 4
		field, delimiter, err := readBoundaryField(reader, quoted)
		if err != nil {
			return boundaryRow{}, err
		}
		fields[index] = field
		if index < len(fields)-1 && delimiter != ',' {
			return boundaryRow{}, fmt.Errorf("field %d must end with comma", index+1)
		}
		if index == len(fields)-1 && delimiter != ')' {
			return boundaryRow{}, errors.New("tuple must end with closing parenthesis")
		}
	}
	return validateBoundary(fields[0], fields[2], fields[3], fields[4])
}

func readBoundaryField(reader *bufio.Reader, quoted bool) (string, byte, error) {
	if err := skipSQLSpace(reader); err != nil {
		return "", 0, err
	}
	if quoted {
		value, err := reader.ReadByte()
		if err != nil {
			return "", 0, err
		}
		if value != '\'' {
			return "", 0, errors.New("expected quoted field")
		}
		var field strings.Builder
		for {
			value, err := reader.ReadByte()
			if err != nil {
				return "", 0, errors.New("unterminated quoted field")
			}
			if value != '\'' {
				field.WriteByte(value)
				continue
			}
			next, err := reader.ReadByte()
			if err != nil {
				return "", 0, errors.New("unterminated quoted field")
			}
			if next == '\'' {
				field.WriteByte('\'')
				continue
			}
			if err := reader.UnreadByte(); err != nil {
				return "", 0, err
			}
			if err := skipSQLSpace(reader); err != nil {
				return "", 0, err
			}
			delimiter, err := reader.ReadByte()
			if err != nil {
				return "", 0, err
			}
			return field.String(), delimiter, nil
		}
	}

	var field strings.Builder
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return "", 0, err
		}
		if value == ',' || value == ')' {
			return strings.TrimSpace(field.String()), value, nil
		}
		field.WriteByte(value)
	}
}

func skipSQLSpace(reader *bufio.Reader) error {
	for {
		value, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if value != ' ' && value != '\t' && value != '\r' && value != '\n' {
			return reader.UnreadByte()
		}
	}
}

func validateBoundary(code, latitude, longitude, path string) (boundaryRow, error) {
	code = strings.TrimSpace(code)
	if !domainlocation.IsValidCode(code) {
		return boundaryRow{}, fmt.Errorf("invalid location code %q", code)
	}
	lat, err := parseCoordinateString(latitude, -90, 90, "latitude")
	if err != nil {
		return boundaryRow{}, err
	}
	lng, err := parseCoordinateString(longitude, -180, 180, "longitude")
	if err != nil {
		return boundaryRow{}, err
	}
	path = strings.TrimSpace(path)
	if err := validateLeafletPath(path); err != nil {
		return boundaryRow{}, err
	}
	return boundaryRow{
		Code:        code,
		Coordinates: domainlocation.Coordinates{Latitude: lat, Longitude: lng},
		LeafletPath: json.RawMessage(path),
	}, nil
}

func parseCoordinate(raw float64, min, max float64, name string) (float64, error) {
	if math.IsNaN(raw) || math.IsInf(raw, 0) || raw < min || raw > max {
		return 0, fmt.Errorf("%s is out of range", name)
	}
	return raw, nil
}

func parseCoordinateString(value string, min, max float64, name string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s is invalid: %w", name, err)
	}
	return parseCoordinate(number, min, max, name)
}

func validateLeafletPath(path string) error {
	if path == "" {
		return errors.New("leaflet_path is required")
	}
	decoder := json.NewDecoder(strings.NewReader(path))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("leaflet_path is invalid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("leaflet_path contains multiple JSON values")
		}
		return fmt.Errorf("leaflet_path is invalid JSON: %w", err)
	}
	return validateLeafletValue(value)
}

func validateLeafletValue(value any) error {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return errors.New("leaflet_path must be a non-empty coordinate array")
	}
	if len(items) == 2 {
		latitude, latOK := items[0].(json.Number)
		longitude, lngOK := items[1].(json.Number)
		if latOK && lngOK {
			if _, err := parseCoordinateString(latitude.String(), -90, 90, "leaflet latitude"); err != nil {
				return err
			}
			if _, err := parseCoordinateString(longitude.String(), -180, 180, "leaflet longitude"); err != nil {
				return err
			}
			return nil
		}
	}
	for _, item := range items {
		if _, ok := item.([]any); !ok {
			return errors.New("leaflet_path contains a non-coordinate value")
		}
		if err := validateLeafletValue(item); err != nil {
			return err
		}
	}
	return nil
}

func isSQLIdentifierByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_'
}

func toLowerASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}
