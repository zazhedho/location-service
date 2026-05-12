package database

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"location-service/utils"
)

func OpenRedis() (*redis.Client, error) {
	var options *redis.Options
	if rawURL := utils.Env("REDIS_URL", ""); rawURL != "" {
		parsed, err := redis.ParseURL(rawURL)
		if err != nil {
			return nil, err
		}
		options = parsed
	}
	if options == nil {
		db, _ := strconv.Atoi(utils.Env("REDIS_DB", "0"))
		options = &redis.Options{
			Addr:         fmt.Sprintf("%s:%s", utils.Env("REDIS_HOST", "localhost"), utils.Env("REDIS_PORT", "6379")),
			Password:     utils.Env("REDIS_PASSWORD", ""),
			DB:           db,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
		}
	}

	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	log.Print("redis connected")
	return client, nil
}
