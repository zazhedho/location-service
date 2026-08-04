package population

import (
	"context"

	domainpopulation "location-service/internal/domain/population"
)

type Repository interface {
	FindPopulationByCode(ctx context.Context, code string) (domainpopulation.Population, error)
}
