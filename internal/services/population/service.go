package population

import (
	"context"
	"database/sql"
	"errors"

	"github.com/redis/go-redis/v9"

	locationcache "location-service/internal/cache/location"
	domainlocation "location-service/internal/domain/location"
	domainpopulation "location-service/internal/domain/population"
	interfacepopulation "location-service/internal/interfaces/population"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type service struct {
	repo  interfacepopulation.Repository
	redis *redis.Client
}

func NewService(repo interfacepopulation.Repository, redisClients ...*redis.Client) interfacepopulation.Service {
	var redisClient *redis.Client
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}
	return &service{repo: repo, redis: redisClient}
}

func (s *service) GetPopulation(ctx context.Context, code string) (domainpopulation.Population, error) {
	if !domainlocation.IsValidCode(code) {
		return domainpopulation.Population{}, invalid("code is invalid")
	}

	key := locationcache.PopulationKey(code)
	if item, ok := locationcache.GetPopulation(ctx, s.redis, key); ok {
		return item, nil
	}

	item, err := s.repo.FindPopulationByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domainpopulation.ErrNotFound) {
		return domainpopulation.Population{}, domainpopulation.ErrNotFound
	}
	if err != nil {
		return domainpopulation.Population{}, err
	}
	locationcache.SetPopulation(ctx, s.redis, key, item)
	return item, nil
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}
