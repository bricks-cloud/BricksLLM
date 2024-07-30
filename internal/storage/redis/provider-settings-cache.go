package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/redis/go-redis/v9"
)

type ProviderSettingsCache struct {
	client *redis.Client
	wt     time.Duration
	rt     time.Duration
}

func NewProviderSettingsCache(c *redis.Client, wt time.Duration, rt time.Duration) *ProviderSettingsCache {
	return &ProviderSettingsCache{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (c *ProviderSettingsCache) Set(pid string, value any, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.wt)
	defer cancel()
	err := c.client.Set(ctx, pid, value, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *ProviderSettingsCache) Delete(pid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.wt)
	defer cancel()
	err := c.client.Del(ctx, pid).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *ProviderSettingsCache) Get(pid string) (*provider.Setting, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.rt)
	defer cancel()

	result := c.client.Get(ctx, pid)
	err := result.Err()
	if err != nil {
		return nil, err
	}

	bs, err := result.Bytes()
	if err != nil {
		return nil, err
	}

	setting := &provider.Setting{}
	err = json.Unmarshal(bs, setting)
	if err != nil {
		return nil, err
	}

	return setting, nil
}
