package island

import (
	"context"
	"database/sql"
	"errors"

	domainisland "location-service/internal/domain/island"
	interfaceisland "location-service/internal/interfaces/island"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) interfaceisland.Repository {
	return &repository{db: db}
}

func (r *repository) ListIslands(ctx context.Context, provinceCode string, page, limit int) (domainisland.Page, error) {
	offset, err := pageOffset(page, limit)
	if err != nil {
		return domainisland.Page{}, err
	}

	var total int
	if provinceCode == "" {
		err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM islands`).Scan(&total)
	} else {
		err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM islands WHERE province_code = $1`, provinceCode).Scan(&total)
	}
	if err != nil {
		return domainisland.Page{}, err
	}

	var rows *sql.Rows
	if provinceCode == "" {
		rows, err = r.db.QueryContext(ctx, `
			SELECT code, province_code, name, latitude, longitude, status, area, notes
			FROM islands
			ORDER BY code
			LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT code, province_code, name, latitude, longitude, status, area, notes
			FROM islands
			WHERE province_code = $1
			ORDER BY code
			LIMIT $2 OFFSET $3`, provinceCode, limit, offset)
	}
	if err != nil {
		return domainisland.Page{}, err
	}
	defer rows.Close()

	items := make([]domainisland.Island, 0)
	for rows.Next() {
		item, err := scanIsland(rows)
		if err != nil {
			return domainisland.Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domainisland.Page{}, err
	}

	return domainisland.Page{
		Items: items,
		Pagination: domainisland.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages(total, limit),
		},
	}, nil
}

func (r *repository) FindIslandByCode(ctx context.Context, code string) (domainisland.Island, error) {
	return scanIsland(r.db.QueryRowContext(ctx, `
		SELECT code, province_code, name, latitude, longitude, status, area, notes
		FROM islands
		WHERE code = $1`, code))
}

type scanner interface {
	Scan(dest ...any) error
}

func scanIsland(row scanner) (domainisland.Island, error) {
	var item domainisland.Island
	var provinceCode, status, notes sql.NullString
	var latitude, longitude, area sql.NullFloat64
	if err := row.Scan(
		&item.Code,
		&provinceCode,
		&item.Name,
		&latitude,
		&longitude,
		&status,
		&area,
		&notes,
	); err != nil {
		return domainisland.Island{}, err
	}
	if provinceCode.Valid {
		item.ProvinceCode = provinceCode.String
	}
	if latitude.Valid {
		item.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		item.Longitude = &longitude.Float64
	}
	if status.Valid {
		item.Status = status.String
	}
	if area.Valid {
		item.Area = &area.Float64
	}
	if notes.Valid {
		item.Notes = notes.String
	}
	return item, nil
}

func pageOffset(page, limit int) (int64, error) {
	if page < 1 || limit < 1 {
		return 0, errors.New("invalid pagination")
	}
	maxInt64 := int64(^uint64(0) >> 1)
	if int64(page-1) > maxInt64/int64(limit) {
		return 0, errors.New("page is too large")
	}
	return int64(page-1) * int64(limit), nil
}

func totalPages(total, limit int) int {
	if total == 0 {
		return 0
	}
	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	return pages
}
