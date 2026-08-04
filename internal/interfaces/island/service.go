package island

import (
	"context"

	domainisland "location-service/internal/domain/island"
)

type Service interface {
	ListIslands(ctx context.Context, provinceCode, page, limit string) (domainisland.Page, error)
	GetIsland(ctx context.Context, code string) (domainisland.Island, error)
}
