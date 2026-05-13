package location

import (
	"context"

	domainlocation "location-service/internal/domain/location"
)

type Repository interface {
	CountStats(ctx context.Context, scope domainlocation.StatsScope) (domainlocation.Stats, error)
	ListProvinces(ctx context.Context) ([]domainlocation.Item, error)
	ListRegencies(ctx context.Context, provinceCode, codeFormat string) ([]domainlocation.Item, error)
	ListDistricts(ctx context.Context, regencyCode, codeFormat string) ([]domainlocation.Item, error)
	ListVillages(ctx context.Context, districtCode, codeFormat string) ([]domainlocation.Item, error)
	Search(ctx context.Context, query string, limit int) ([]domainlocation.Item, error)
}
