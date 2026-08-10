package location

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"location-service/internal/boundary"
	domainlocation "location-service/internal/domain/location"
	interfacelocation "location-service/internal/interfaces/location"
	"location-service/pkg/storage"
)

type repository struct {
	db      *sql.DB
	storage storage.Provider
}

func NewRepository(db *sql.DB, providers ...storage.Provider) interfacelocation.Repository {
	var provider storage.Provider
	if len(providers) > 0 {
		provider = providers[0]
	}
	return &repository{db: db, storage: provider}
}

func (r *repository) CountStats(ctx context.Context, scope domainlocation.StatsScope) (domainlocation.Stats, error) {
	query, args := statsQuery(scope)

	var stats domainlocation.Stats
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.Raw,
		&stats.Provinces,
		&stats.Regencies,
		&stats.Districts,
		&stats.Villages,
	)
	if err != nil && isTransient(err) {
		err = r.db.QueryRowContext(ctx, query, args...).Scan(
			&stats.Raw,
			&stats.Provinces,
			&stats.Regencies,
			&stats.Districts,
			&stats.Villages,
		)
	}
	if err != nil {
		return domainlocation.Stats{}, err
	}
	stats.Total = stats.Provinces + stats.Regencies + stats.Districts + stats.Villages
	return stats, nil
}

func statsQuery(scope domainlocation.StatsScope) (string, []any) {
	switch scope.Level {
	case "province":
		return `
		SELECT
			(SELECT COUNT(*) FROM raw_locations WHERE code = $1 OR code LIKE $1 || '.%'),
			(SELECT COUNT(*) FROM provinces WHERE code = $1),
			(SELECT COUNT(*) FROM regencies WHERE province_code = $1),
			(SELECT COUNT(*) FROM districts WHERE province_code = $1),
			(SELECT COUNT(*) FROM villages WHERE province_code = $1)`, []any{scope.Code}
	case "regency":
		return `
		SELECT
			(SELECT COUNT(*) FROM raw_locations WHERE code = $1 OR code LIKE $1 || '.%'),
			0,
			(SELECT COUNT(*) FROM regencies WHERE code = $1),
			(SELECT COUNT(*) FROM districts WHERE regency_code = $1),
			(SELECT COUNT(*) FROM villages WHERE regency_code = $1)`, []any{scope.Code}
	case "district":
		return `
		SELECT
			(SELECT COUNT(*) FROM raw_locations WHERE code = $1 OR code LIKE $1 || '.%'),
			0,
			0,
			(SELECT COUNT(*) FROM districts WHERE code = $1),
			(SELECT COUNT(*) FROM villages WHERE district_code = $1)`, []any{scope.Code}
	default:
		return `
		SELECT
			(SELECT COUNT(*) FROM raw_locations),
			(SELECT COUNT(*) FROM provinces),
			(SELECT COUNT(*) FROM regencies),
			(SELECT COUNT(*) FROM districts),
			(SELECT COUNT(*) FROM villages)`, nil
	}
}

func (r *repository) ListProvinces(ctx context.Context) ([]domainlocation.Item, error) {
	return r.queryLocations(ctx, `SELECT code, code AS full_code, name, 'province' AS level, '' AS parent_code, NULL AS postal_code FROM provinces ORDER BY code`)
}

func (r *repository) ListRegencies(ctx context.Context, provinceCode, codeFormat string) ([]domainlocation.Item, error) {
	codeExpr := codeExpression(codeFormat)
	query := fmt.Sprintf(`SELECT %s AS code, code AS full_code, name, 'regency' AS level, province_code AS parent_code, NULL AS postal_code FROM regencies WHERE province_code = $1 ORDER BY code`, codeExpr)
	return r.queryLocations(ctx, query, provinceCode)
}

func (r *repository) ListDistricts(ctx context.Context, regencyCode, codeFormat string) ([]domainlocation.Item, error) {
	codeExpr := codeExpression(codeFormat)
	query := fmt.Sprintf(`SELECT %s AS code, code AS full_code, name, 'district' AS level, regency_code AS parent_code, NULL AS postal_code FROM districts WHERE regency_code = $1 ORDER BY code`, codeExpr)
	return r.queryLocations(ctx, query, regencyCode)
}

func (r *repository) ListVillages(ctx context.Context, districtCode, codeFormat string) ([]domainlocation.Item, error) {
	codeExpr := codeExpression(codeFormat)
	query := fmt.Sprintf(`SELECT %s AS code, code AS full_code, name, 'village' AS level, district_code AS parent_code, postal_code FROM villages WHERE district_code = $1 ORDER BY code`, codeExpr)
	return r.queryLocations(ctx, query, districtCode)
}

func (r *repository) Search(ctx context.Context, query string, limit int) ([]domainlocation.Item, error) {
	return r.queryLocations(ctx, `
		SELECT l.code, l.code AS full_code, l.name,
		       CASE l.level WHEN 1 THEN 'province' WHEN 2 THEN 'regency' WHEN 3 THEN 'district' ELSE 'village' END AS level,
		       '' AS parent_code, v.postal_code
		FROM raw_locations l
		LEFT JOIN villages v ON v.code = l.code
		WHERE l.name ILIKE '%' || $1 || '%'
		ORDER BY l.level, l.code
		LIMIT $2`, query, limit)
}

func (r *repository) SearchByPostalCode(ctx context.Context, postalCode string) ([]domainlocation.PostalLocation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT v.postal_code,
		       v.code, v.name,
		       d.code, d.name,
		       rg.code, rg.name,
		       p.code, p.name
		FROM villages v
		JOIN districts d ON d.code = v.district_code
		JOIN regencies rg ON rg.code = v.regency_code
		JOIN provinces p ON p.code = v.province_code
		WHERE v.postal_code = $1
		ORDER BY p.code, rg.code, d.code, v.code`, postalCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domainlocation.PostalLocation, 0)
	for rows.Next() {
		var item domainlocation.PostalLocation
		if err := rows.Scan(
			&item.PostalCode,
			&item.Village.Code, &item.Village.Name,
			&item.District.Code, &item.District.Name,
			&item.Regency.Code, &item.Regency.Name,
			&item.Province.Code, &item.Province.Name,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *repository) GetDetail(ctx context.Context, code string) (domainlocation.Detail, error) {
	var detail domainlocation.Detail
	var level int
	var postalCode sql.NullString
	var lat, lng sql.NullFloat64
	var hasBoundary bool
	err := r.db.QueryRowContext(ctx, `
		SELECT l.code, l.code, l.name, l.level,
		       v.postal_code,
		       b.centroid_lat, b.centroid_lng, b.code IS NOT NULL
		FROM raw_locations l
		LEFT JOIN villages v ON v.code = l.code
		LEFT JOIN location_boundaries b ON b.code = l.code
		WHERE l.code = $1`, code).Scan(
		&detail.Code,
		&detail.FullCode,
		&detail.Name,
		&level,
		&postalCode,
		&lat,
		&lng,
		&hasBoundary,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domainlocation.Detail{}, domainlocation.ErrNotFound
		}
		return domainlocation.Detail{}, err
	}
	detail.Level = levelName(level)
	if postalCode.Valid {
		detail.PostalCode = postalCode.String
	}
	if index := strings.LastIndexByte(detail.Code, '.'); index > 0 {
		detail.ParentCode = detail.Code[:index]
	}
	detail.HasBoundary = hasBoundary
	if lat.Valid && lng.Valid {
		detail.Coordinates = &domainlocation.Coordinates{Latitude: lat.Float64, Longitude: lng.Float64}
	}
	return detail, nil
}

func (r *repository) GetBoundary(ctx context.Context, code string) (domainlocation.Boundary, error) {
	var boundary domainlocation.Boundary
	var objectKey sql.NullString
	var path []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT b.code, l.name, b.centroid_lat, b.centroid_lng, b.object_key, b.leaflet_path
		FROM location_boundaries b
		JOIN raw_locations l ON l.code = b.code
		WHERE b.code = $1`, code).Scan(
		&boundary.Code,
		&boundary.Name,
		&boundary.Latitude,
		&boundary.Longitude,
		&objectKey,
		&path,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domainlocation.Boundary{}, domainlocation.ErrBoundaryNotFound
		}
		return domainlocation.Boundary{}, err
	}
	if objectKey.Valid && objectKey.String != "" {
		if r.storage == nil {
			return domainlocation.Boundary{}, fmt.Errorf("boundary storage is not configured")
		}
		path, err = loadBoundaryPayload(ctx, r.storage, objectKey.String)
		if err != nil {
			return domainlocation.Boundary{}, err
		}
	}
	if len(path) == 0 {
		return domainlocation.Boundary{}, domainlocation.ErrBoundaryNotFound
	}
	boundary.LeafletPath = path
	return boundary, nil
}

func loadBoundaryPayload(ctx context.Context, provider storage.Provider, key string) ([]byte, error) {
	reader, err := provider.Download(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotFound) {
			return nil, domainlocation.ErrBoundaryNotFound
		}
		return nil, err
	}
	defer reader.Close()
	payload, err := boundary.DecodeBoundaryPayload(reader)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func levelName(level int) string {
	switch level {
	case 1:
		return "province"
	case 2:
		return "regency"
	case 3:
		return "district"
	default:
		return "village"
	}
}
