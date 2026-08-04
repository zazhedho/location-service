package locationcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	domainisland "location-service/internal/domain/island"
)

func IslandListKey(provinceCode string, page, limit int) string {
	return fmt.Sprintf("%sislands:list:%s:%d:%d", prefix, clean(provinceCode), page, limit)
}

func IslandDetailKey(code string) string {
	return fmt.Sprintf("%sislands:detail:%s", prefix, clean(code))
}

func GetIslandPage(ctx context.Context, client *redis.Client, key string) (domainisland.Page, bool) {
	if client == nil {
		return domainisland.Page{}, false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("location cache get failed; key=%s; err=%v", key, err)
		}
		return domainisland.Page{}, false
	}

	var page domainisland.Page
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		log.Printf("location cache unmarshal failed; key=%s; err=%v", key, err)
		return domainisland.Page{}, false
	}
	return page, true
}

func SetIslandPage(ctx context.Context, client *redis.Client, key string, page domainisland.Page) {
	if client == nil {
		return
	}
	setIslandJSON(ctx, client, key, page)
}

func GetIsland(ctx context.Context, client *redis.Client, key string) (domainisland.Island, bool) {
	if client == nil {
		return domainisland.Island{}, false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("location cache get failed; key=%s; err=%v", key, err)
		}
		return domainisland.Island{}, false
	}

	var island domainisland.Island
	if err := json.Unmarshal([]byte(raw), &island); err != nil {
		log.Printf("location cache unmarshal failed; key=%s; err=%v", key, err)
		return domainisland.Island{}, false
	}
	return island, true
}

func SetIsland(ctx context.Context, client *redis.Client, key string, island domainisland.Island) {
	if client == nil {
		return
	}
	setIslandJSON(ctx, client, key, island)
}

func setIslandJSON(ctx context.Context, client *redis.Client, key string, value any) {
	payload, err := json.Marshal(value)
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
