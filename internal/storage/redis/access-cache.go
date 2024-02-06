package redis

import (
	"context"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/redis/go-redis/v9"
)

type AccessCache struct {
	client *redis.Client
	wt     time.Duration
	rt     time.Duration
}

func NewAccessCache(c *redis.Client, wt time.Duration, rt time.Duration) *AccessCache {
	return &AccessCache{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (ac *AccessCache) Set(key string, timeUnit key.TimeUnit) error {
	ttl, err := getCounterTtl(timeUnit)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), ac.wt)
	defer cancel()
	err = ac.client.Set(ctx, key, true, ttl.Sub(time.Now())).Err()
	if err != nil {
		return err
	}

	return nil
}

func (ac *AccessCache) GetAccessStatus(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), ac.rt)
	defer cancel()

	result := ac.client.Get(ctx, key)

	return result.Err() != redis.Nil
}
