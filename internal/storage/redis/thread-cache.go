package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type ThreadCache struct {
	client *redis.Client
	wt     time.Duration
	rt     time.Duration
}

func NewThreadCache(c *redis.Client, wt time.Duration, rt time.Duration) *ThreadCache {
	return &ThreadCache{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (ac *ThreadCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ac.wt)
	defer cancel()
	err := ac.client.Del(ctx, key).Err()
	if err != nil {
		return err
	}

	return nil
}

func (ac *ThreadCache) Set(key string, dur time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), ac.wt)
	defer cancel()
	err := ac.client.Set(ctx, key, true, dur).Err()
	if err != nil {
		return err
	}

	return nil
}

func (ac *ThreadCache) GetThreadStatus(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), ac.rt)
	defer cancel()

	result := ac.client.Get(ctx, key)

	return result.Err() != redis.Nil
}
