package location

import (
	"errors"
	domainlocation "location-service/internal/domain/location"
	"strings"
)

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

func resolveStatsScope(provinceCode, regencyCode, districtCode string) (domainlocation.StatsScope, error) {
	provinceCode = strings.TrimSpace(provinceCode)
	regencyCode = strings.TrimSpace(regencyCode)
	districtCode = strings.TrimSpace(districtCode)

	if districtCode != "" {
		if strings.Count(districtCode, ".") == 2 {
			return domainlocation.StatsScope{Level: "district", Code: districtCode}, nil
		}
		resolvedRegencyCode, err := resolveChildCode(provinceCode, regencyCode, "regency_code")
		if err != nil {
			return domainlocation.StatsScope{}, err
		}
		return domainlocation.StatsScope{Level: "district", Code: resolvedRegencyCode + "." + districtCode}, nil
	}
	if regencyCode != "" {
		if strings.Count(regencyCode, ".") == 1 {
			return domainlocation.StatsScope{Level: "regency", Code: regencyCode}, nil
		}
		resolvedRegencyCode, err := resolveChildCode(provinceCode, regencyCode, "regency_code")
		if err != nil {
			return domainlocation.StatsScope{}, err
		}
		return domainlocation.StatsScope{Level: "regency", Code: resolvedRegencyCode}, nil
	}
	if provinceCode != "" {
		return domainlocation.StatsScope{Level: "province", Code: provinceCode}, nil
	}
	return domainlocation.StatsScope{}, nil
}
