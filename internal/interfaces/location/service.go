package location

import (
	"context"

	domainlocation "location-service/internal/domain/location"
)

type Service interface {
	ListProvinces(ctx context.Context) ([]domainlocation.Item, error)
	ListRegencies(ctx context.Context, provinceCode, codeFormat string) ([]domainlocation.Item, error)
	ListDistricts(ctx context.Context, provinceCode, regencyCode, codeFormat string) ([]domainlocation.Item, error)
	ListVillages(ctx context.Context, provinceCode, regencyCode, districtCode, codeFormat string) ([]domainlocation.Item, error)
	Search(ctx context.Context, query string, limit string) ([]domainlocation.Item, error)
}
