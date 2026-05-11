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

func codeExpression(codeFormat string) string {
	if codeFormat == "short" {
		return "short_code"
	}
	return "code"
}

func (r *repository) queryLocations(ctx context.Context, query string, args ...any) ([]domainlocation.Item, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domainlocation.Item, 0)
	for rows.Next() {
		var item domainlocation.Item
		if err := rows.Scan(&item.Code, &item.FullCode, &item.Name, &item.Level, &item.ParentCode); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
