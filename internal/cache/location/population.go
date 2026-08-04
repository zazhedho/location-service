package locationcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	domainpopulation "location-service/internal/domain/population"
)

func PopulationKey(code string) string {
	return fmt.Sprintf("%spopulation:%s", prefix, clean(code))
}

func GetPopulation(ctx context.Context, client *redis.Client, key string) (domainpopulation.Population, bool) {
	if client == nil {
		return domainpopulation.Population{}, false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			log.Printf("location cache get failed; key=%s; err=%v", key, err)
		}
		return domainpopulation.Population{}, false
	}

	var item domainpopulation.Population
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		log.Printf("location cache unmarshal failed; key=%s; err=%v", key, err)
		return domainpopulation.Population{}, false
	}
	return item, true
}

func SetPopulation(ctx context.Context, client *redis.Client, key string, item domainpopulation.Population) {
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
