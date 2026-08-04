package area

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/redis/go-redis/v9"

	locationcache "location-service/internal/cache/location"
	domainarea "location-service/internal/domain/area"
	domainlocation "location-service/internal/domain/location"
	interfacearea "location-service/internal/interfaces/area"
)

type service struct {
	repo  interfacearea.Repository
	redis *redis.Client
}

func NewService(repo interfacearea.Repository, redisClients ...*redis.Client) interfacearea.Service {
	var redisClient *redis.Client
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}
	return &service{repo: repo, redis: redisClient}
}

func (s *service) Area(ctx context.Context, code string) (domainarea.Area, error) {
	code, err := normalizeCode(code)
	if err != nil {
		return domainarea.Area{}, err
	}

	key := locationcache.AreaKey(code)
	if item, ok, missing := locationcache.GetArea(ctx, s.redis, key); ok {
		if missing {
			return domainarea.Area{}, domainarea.ErrNotFound
		}
		return item, nil
	}

	item, err := s.repo.FindAreaByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domainarea.ErrNotFound) {
		locationcache.SetAreaMissing(ctx, s.redis, key)
		return domainarea.Area{}, domainarea.ErrNotFound
	}
	if err != nil {
		return domainarea.Area{}, err
	}
	locationcache.SetArea(ctx, s.redis, key, item)
	return item, nil
}

func normalizeCode(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", domainarea.ErrCodeRequired
	}
	if !domainlocation.IsValidCode(value) {
		return "", domainarea.ErrCodeInvalid
	}
	return value, nil
}
