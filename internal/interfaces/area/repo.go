package area

import (
	"context"

	domainarea "location-service/internal/domain/area"
)

type Repository interface {
	FindAreaByCode(ctx context.Context, code string) (domainarea.Area, error)
}
