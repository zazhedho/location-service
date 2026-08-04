package area

import (
	"context"

	domainarea "location-service/internal/domain/area"
)

type Service interface {
	Area(ctx context.Context, code string) (domainarea.Area, error)
}
