package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/redis/go-redis/v9"
)

type KeysCache struct {
	client *redis.Client
	wt     time.Duration
	rt     time.Duration
}

func NewKeysCache(c *redis.Client, wt time.Duration, rt time.Duration) *KeysCache {
	return &KeysCache{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (c *KeysCache) Set(pid string, value any, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.wt)
	defer cancel()
	err := c.client.Set(ctx, pid, value, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *KeysCache) Delete(pid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.wt)
	defer cancel()
	err := c.client.Del(ctx, pid).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *KeysCache) Get(pid string) (*key.ResponseKey, error) {
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

	k := &key.ResponseKey{}
	err = json.Unmarshal(bs, k)
	if err != nil {
		return nil, err
	}

	return k, nil
}
