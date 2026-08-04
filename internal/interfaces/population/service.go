package population

import (
	"context"

	domainpopulation "location-service/internal/domain/population"
)

type Service interface {
	GetPopulation(ctx context.Context, code string) (domainpopulation.Population, error)
}
