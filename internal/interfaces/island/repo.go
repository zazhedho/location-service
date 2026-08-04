package island

import (
	"context"

	domainisland "location-service/internal/domain/island"
)

type Repository interface {
	ListIslands(ctx context.Context, provinceCode string, page, limit int) (domainisland.Page, error)
	FindIslandByCode(ctx context.Context, code string) (domainisland.Island, error)
}
