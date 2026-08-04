package locationcache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	domainarea "location-service/internal/domain/area"
)

const areaNegativeTTL = time.Minute

func AreaKey(code string) string {
	return "location:area:" + clean(code)
}

func GetArea(ctx context.Context, client *redis.Client, key string) (domainarea.Area, bool, bool) {
	if client == nil {
		return domainarea.Area{}, false, false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("location cache get failed; key=%s; err=%v", key, err)
		}
		return domainarea.Area{}, false, false
	}
	if raw == "__missing_area__" {
		return domainarea.Area{}, true, true
	}

	var item domainarea.Area
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		log.Printf("location cache unmarshal failed; key=%s; err=%v", key, err)
		return domainarea.Area{}, false, false
	}
	return item, true, false
}

func SetArea(ctx context.Context, client *redis.Client, key string, item domainarea.Area) {
	if client == nil {
		return
	}
	payload, err := json.Marshal(item)
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

func SetAreaMissing(ctx context.Context, client *redis.Client, key string) {
	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Set(ctx, key, "__missing_area__", areaNegativeTTL).Err(); err != nil {
		log.Printf("location cache set failed; key=%s; err=%v", key, err)
	}
}
