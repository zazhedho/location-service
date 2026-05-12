package location

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	locationcache "location-service/internal/cache/location"
	domainlocation "location-service/internal/domain/location"
	interfacelocation "location-service/internal/interfaces/location"
)

type service struct {
	repo  interfacelocation.Repository
	redis *redis.Client
}

func NewService(repo interfacelocation.Repository, redisClients ...*redis.Client) interfacelocation.Service {
	var redisClient *redis.Client
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}
	return &service{repo: repo, redis: redisClient}
}

func (s *service) ListProvinces(ctx context.Context) ([]domainlocation.Item, error) {
	key := locationcache.ProvinceKey()
	if items, ok := locationcache.Get(ctx, s.redis, key); ok {
		return items, nil
	}
	items, err := s.repo.ListProvinces(ctx)
	if err != nil {
		return nil, err
	}
	locationcache.Set(ctx, s.redis, key, items)
	return items, nil
}

func (s *service) ListRegencies(ctx context.Context, provinceCode, codeFormat string) ([]domainlocation.Item, error) {
	provinceCode = strings.TrimSpace(provinceCode)
	if provinceCode == "" {
		return nil, errors.New("province_code is required")
	}
	codeFormat = normalizeCodeFormat(codeFormat)
	key := locationcache.RegencyKey(provinceCode, codeFormat)
	if items, ok := locationcache.Get(ctx, s.redis, key); ok {
		return items, nil
	}
	items, err := s.repo.ListRegencies(ctx, provinceCode, codeFormat)
	if err != nil {
		return nil, err
	}
	locationcache.Set(ctx, s.redis, key, items)
	return items, nil
}

func (s *service) ListDistricts(ctx context.Context, provinceCode, regencyCode, codeFormat string) ([]domainlocation.Item, error) {
	resolvedRegencyCode, err := resolveChildCode(provinceCode, regencyCode, "regency_code")
	if err != nil {
		return nil, err
	}
	codeFormat = normalizeCodeFormat(codeFormat)
	key := locationcache.DistrictKey(resolvedRegencyCode, codeFormat)
	if items, ok := locationcache.Get(ctx, s.redis, key); ok {
		return items, nil
	}
	items, err := s.repo.ListDistricts(ctx, resolvedRegencyCode, codeFormat)
	if err != nil {
		return nil, err
	}
	locationcache.Set(ctx, s.redis, key, items)
	return items, nil
}

func (s *service) ListVillages(ctx context.Context, provinceCode, regencyCode, districtCode, codeFormat string) ([]domainlocation.Item, error) {
	districtCode = strings.TrimSpace(districtCode)
	if districtCode == "" {
		return nil, errors.New("district_code is required")
	}
	codeFormat = normalizeCodeFormat(codeFormat)
	if strings.Count(districtCode, ".") == 2 {
		return s.listVillagesByDistrict(ctx, districtCode, codeFormat)
	}
	resolvedRegencyCode, err := resolveChildCode(provinceCode, regencyCode, "regency_code")
	if err != nil {
		return nil, err
	}
	return s.listVillagesByDistrict(ctx, resolvedRegencyCode+"."+districtCode, codeFormat)
}

func (s *service) Search(ctx context.Context, query string, limit string) ([]domainlocation.Item, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("q is required")
	}
	parsedLimit := 50
	if raw := strings.TrimSpace(limit); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 || value > 500 {
			return nil, errors.New("limit must be a number between 1 and 500")
		}
		parsedLimit = value
	}
	key := locationcache.SearchKey(query, parsedLimit)
	if items, ok := locationcache.Get(ctx, s.redis, key); ok {
		return items, nil
	}
	items, err := s.repo.Search(ctx, query, parsedLimit)
	if err != nil {
		return nil, err
	}
	locationcache.Set(ctx, s.redis, key, items)
	return items, nil
}

func (s *service) listVillagesByDistrict(ctx context.Context, districtCode, codeFormat string) ([]domainlocation.Item, error) {
	key := locationcache.VillageKey(districtCode, codeFormat)
	if items, ok := locationcache.Get(ctx, s.redis, key); ok {
		return items, nil
	}
	items, err := s.repo.ListVillages(ctx, districtCode, codeFormat)
	if err != nil {
		return nil, err
	}
	locationcache.Set(ctx, s.redis, key, items)
	return items, nil
}

func normalizeCodeFormat(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "short") {
		return "short"
	}
	return "full"
}

func resolveChildCode(parentCode, childCode, childName string) (string, error) {
	childCode = strings.TrimSpace(childCode)
	if childCode == "" {
		return "", errors.New(childName + " is required")
	}
	if strings.Contains(childCode, ".") {
		return childCode, nil
	}
	parentCode = strings.TrimSpace(parentCode)
	if parentCode == "" {
		return "", errors.New("province_code is required when " + childName + " is short")
	}
	return parentCode + "." + childCode, nil
}
