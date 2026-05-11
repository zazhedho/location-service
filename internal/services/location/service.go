package location

import (
	"context"
	"errors"
	"strconv"
	"strings"

	domainlocation "location-service/internal/domain/location"
	interfacelocation "location-service/internal/interfaces/location"
)

type service struct {
	repo interfacelocation.Repository
}

func NewService(repo interfacelocation.Repository) interfacelocation.Service {
	return &service{repo: repo}
}

func (s *service) ListProvinces(ctx context.Context) ([]domainlocation.Item, error) {
	return s.repo.ListProvinces(ctx)
}

func (s *service) ListRegencies(ctx context.Context, provinceCode, codeFormat string) ([]domainlocation.Item, error) {
	provinceCode = strings.TrimSpace(provinceCode)
	if provinceCode == "" {
		return nil, errors.New("province_code is required")
	}
	return s.repo.ListRegencies(ctx, provinceCode, normalizeCodeFormat(codeFormat))
}

func (s *service) ListDistricts(ctx context.Context, provinceCode, regencyCode, codeFormat string) ([]domainlocation.Item, error) {
	resolvedRegencyCode, err := resolveChildCode(provinceCode, regencyCode, "regency_code")
	if err != nil {
		return nil, err
	}
	return s.repo.ListDistricts(ctx, resolvedRegencyCode, normalizeCodeFormat(codeFormat))
}

func (s *service) ListVillages(ctx context.Context, provinceCode, regencyCode, districtCode, codeFormat string) ([]domainlocation.Item, error) {
	districtCode = strings.TrimSpace(districtCode)
	if districtCode == "" {
		return nil, errors.New("district_code is required")
	}
	if strings.Count(districtCode, ".") == 2 {
		return s.repo.ListVillages(ctx, districtCode, normalizeCodeFormat(codeFormat))
	}
	resolvedRegencyCode, err := resolveChildCode(provinceCode, regencyCode, "regency_code")
	if err != nil {
		return nil, err
	}
	return s.repo.ListVillages(ctx, resolvedRegencyCode+"."+districtCode, normalizeCodeFormat(codeFormat))
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
	return s.repo.Search(ctx, query, parsedLimit)
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
