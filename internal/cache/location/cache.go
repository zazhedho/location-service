package locationcache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	domainlocation "location-service/internal/domain/location"
	"location-service/utils"
)

const (
	defaultTTL = 180 * 24 * time.Hour
	prefix     = "location:"
)

func TTL() time.Duration {
	raw := strings.TrimSpace(utils.Env("LOCATION_CACHE_TTL", ""))
	if raw == "" {
		return defaultTTL
	}
	if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
		return parsed
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultTTL
}

func ProvinceKey() string {
	return prefix + "provinces"
}

func RegencyKey(provinceCode, codeFormat string) string {
	return fmt.Sprintf("%sregencies:%s:%s", prefix, clean(provinceCode), clean(codeFormat))
}

func DistrictKey(regencyCode, codeFormat string) string {
	return fmt.Sprintf("%sdistricts:%s:%s", prefix, clean(regencyCode), clean(codeFormat))
}

func VillageKey(districtCode, codeFormat string) string {
	return fmt.Sprintf("%svillages:%s:%s", prefix, clean(districtCode), clean(codeFormat))
}

func SearchKey(query string, limit int) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(query))))
	return fmt.Sprintf("%ssearch:%s:%d", prefix, hex.EncodeToString(sum[:]), limit)
}

func Get(ctx context.Context, client *redis.Client, key string) ([]domainlocation.Item, bool) {
	if client == nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("location cache get failed; key=%s; err=%v", key, err)
		}
		return nil, false
	}

	var items []domainlocation.Item
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		log.Printf("location cache unmarshal failed; key=%s; err=%v", key, err)
		return nil, false
	}
	return items, true
}

func Set(ctx context.Context, client *redis.Client, key string, items []domainlocation.Item) {
	if client == nil {
		return
	}

	payload, err := json.Marshal(items)
	if err != nil {
		log.Printf("location cache marshal failed; key=%s; err=%v", key, err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Set(ctx, key, payload, TTL()).Err(); err != nil {
		log.Printf("location cache set failed; key=%s; err=%v", key, err)
	}
}

func clean(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}
