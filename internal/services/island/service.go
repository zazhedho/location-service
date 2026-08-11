package island

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	locationcache "location-service/internal/cache/location"
	domainisland "location-service/internal/domain/island"
	interfaceisland "location-service/internal/interfaces/island"
)

const (
	DefaultPage  = 1
	DefaultLimit = 50
	MaxPage      = 1000
	MaxLimit     = 500
)

var (
	ErrNotFound       = errors.New("island not found")
	provinceCodeRegex = regexp.MustCompile(`^[0-9]{2}$`)
	islandCodeRegex   = regexp.MustCompile(`^[0-9]{2}\.[0-9]{2}\.[0-9]{5}$`)
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type service struct {
	repo  interfaceisland.Repository
	redis *redis.Client
}

func NewService(repo interfaceisland.Repository, redisClients ...*redis.Client) interfaceisland.Service {
	var redisClient *redis.Client
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}
	return &service{repo: repo, redis: redisClient}
}

func (s *service) ListIslands(ctx context.Context, provinceCode, page, limit string) (domainisland.Page, error) {
	provinceCode = strings.TrimSpace(provinceCode)
	if provinceCode != "" && !provinceCodeRegex.MatchString(provinceCode) {
		return domainisland.Page{}, invalid("province_code must be a two-digit code")
	}

	pageNumber, err := parsePage(page)
	if err != nil {
		return domainisland.Page{}, err
	}
	limitNumber, err := parseLimit(limit)
	if err != nil {
		return domainisland.Page{}, err
	}
	if _, err := pageOffset(pageNumber, limitNumber); err != nil {
		return domainisland.Page{}, invalid(err.Error())
	}

	key := locationcache.IslandListKey(provinceCode, pageNumber, limitNumber)
	if pageData, ok := locationcache.GetIslandPage(ctx, s.redis, key); ok {
		return pageData, nil
	}

	pageData, err := s.repo.ListIslands(ctx, provinceCode, pageNumber, limitNumber)
	if err != nil {
		return domainisland.Page{}, err
	}
	locationcache.SetIslandPage(ctx, s.redis, key, pageData)
	return pageData, nil
}

func (s *service) GetIsland(ctx context.Context, code string) (domainisland.Island, error) {
	code = strings.TrimSpace(code)
	if !islandCodeRegex.MatchString(code) {
		return domainisland.Island{}, invalid("code must match XX.XX.XXXXX")
	}

	key := locationcache.IslandDetailKey(code)
	if item, ok := locationcache.GetIsland(ctx, s.redis, key); ok {
		return item, nil
	}

	item, err := s.repo.FindIslandByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return domainisland.Island{}, ErrNotFound
	}
	if err != nil {
		return domainisland.Island{}, err
	}
	locationcache.SetIsland(ctx, s.redis, key, item)
	return item, nil
}

func invalid(message string) error {
	return &ValidationError{Message: message}
}

func parsePage(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultPage, nil
	}
	value, err := parseDigits(raw)
	if err != nil || value < 1 || value > MaxPage {
		return 0, invalid("page must be a number between 1 and 1000")
	}
	return value, nil
}

func parseLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultLimit, nil
	}
	value, err := parseDigits(raw)
	if err != nil || value < 1 || value > MaxLimit {
		return 0, invalid("limit must be a number between 1 and 500")
	}
	return value, nil
}

func parseDigits(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	for _, char := range raw {
		if char < '0' || char > '9' {
			return 0, errors.New("not a decimal integer")
		}
	}
	return strconv.Atoi(raw)
}

func pageOffset(page, limit int) (int64, error) {
	if page < 1 || limit < 1 {
		return 0, errors.New("invalid pagination")
	}
	maxInt64 := int64(^uint64(0) >> 1)
	if int64(page-1) > maxInt64/int64(limit) {
		return 0, errors.New("page is too large")
	}
	return int64(page-1) * int64(limit), nil
}
