package area

import (
	"context"
	"database/sql"

	domainarea "location-service/internal/domain/area"
	interfacearea "location-service/internal/interfaces/area"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) interfacearea.Repository {
	return &repository{db: db}
}

func (r *repository) FindAreaByCode(ctx context.Context, code string) (domainarea.Area, error) {
	var item domainarea.Area
	err := r.db.QueryRowContext(ctx, `
		SELECT l.code, l.name, a.area_km2, a.source,
		       a.reference_date::text, a.imported_at
		FROM location_areas a
		JOIN raw_locations l ON l.code = a.code
		WHERE a.code = $1`, code).Scan(
		&item.Code,
		&item.Name,
		&item.AreaKM2,
		&item.Source,
		&item.ReferenceDate,
		&item.ImportedAt,
	)
	if err == sql.ErrNoRows {
		return domainarea.Area{}, domainarea.ErrNotFound
	}
	if err != nil {
		return domainarea.Area{}, err
	}
	return item, nil
}
