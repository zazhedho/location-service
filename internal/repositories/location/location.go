package location

import (
	"context"
	"database/sql"
	"fmt"
	domainlocation "location-service/internal/domain/location"
	interfacelocation "location-service/internal/interfaces/location"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) interfacelocation.Repository {
	return &repository{db: db}
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
	return r.queryLocations(ctx, `SELECT code, code AS full_code, name, 'province' AS level, '' AS parent_code FROM provinces ORDER BY code`)
}

func (r *repository) ListRegencies(ctx context.Context, provinceCode, codeFormat string) ([]domainlocation.Item, error) {
	codeExpr := codeExpression(codeFormat)
	query := fmt.Sprintf(`SELECT %s AS code, code AS full_code, name, 'regency' AS level, province_code AS parent_code FROM regencies WHERE province_code = $1 ORDER BY code`, codeExpr)
	return r.queryLocations(ctx, query, provinceCode)
}

func (r *repository) ListDistricts(ctx context.Context, regencyCode, codeFormat string) ([]domainlocation.Item, error) {
	codeExpr := codeExpression(codeFormat)
	query := fmt.Sprintf(`SELECT %s AS code, code AS full_code, name, 'district' AS level, regency_code AS parent_code FROM districts WHERE regency_code = $1 ORDER BY code`, codeExpr)
	return r.queryLocations(ctx, query, regencyCode)
}

func (r *repository) ListVillages(ctx context.Context, districtCode, codeFormat string) ([]domainlocation.Item, error) {
	codeExpr := codeExpression(codeFormat)
	query := fmt.Sprintf(`SELECT %s AS code, code AS full_code, name, 'village' AS level, district_code AS parent_code FROM villages WHERE district_code = $1 ORDER BY code`, codeExpr)
	return r.queryLocations(ctx, query, districtCode)
}

func (r *repository) Search(ctx context.Context, query string, limit int) ([]domainlocation.Item, error) {
	return r.queryLocations(ctx, `
		SELECT code, code AS full_code, name,
		       CASE level WHEN 1 THEN 'province' WHEN 2 THEN 'regency' WHEN 3 THEN 'district' ELSE 'village' END AS level,
		       '' AS parent_code
		FROM raw_locations
		WHERE name ILIKE '%' || $1 || '%'
		ORDER BY level, code
		LIMIT $2`, query, limit)
}
