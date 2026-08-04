package population

import (
	"context"
	"database/sql"

	domainpopulation "location-service/internal/domain/population"
	interfacepopulation "location-service/internal/interfaces/population"
)

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) interfacepopulation.Repository {
	return &repository{db: db}
}

func (r *repository) FindPopulationByCode(ctx context.Context, code string) (domainpopulation.Population, error) {
	var item domainpopulation.Population
	err := r.db.QueryRowContext(ctx, `
		SELECT p.code, l.name, p.male, p.female, p.total,
		       p.source, to_char(p.reference_date, 'YYYY-MM-DD'), p.imported_at
		FROM location_population p
		JOIN raw_locations l ON l.code = p.code
		WHERE p.code = $1`, code).Scan(
		&item.Code,
		&item.Name,
		&item.Male,
		&item.Female,
		&item.Total,
		&item.Source,
		&item.ReferenceDate,
		&item.ImportedAt,
	)
	return item, err
}
