package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	client *redis.Client
	wt     time.Duration
	rt     time.Duration
}

func NewStore(c *redis.Client, wt time.Duration, rt time.Duration) *Store {
	return &Store{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (s *Store) IncrementCounter(prefix string, keyId string, incr int64) error {
	redisKey := prefix + "-" + keyId
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	return s.client.IncrBy(ctxTimeout, redisKey, incr).Err()
}

func (s *Store) DeleteCounter(prefix string, keyId string) error {
	redisKey := prefix + "-" + keyId
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	return s.client.Del(ctxTimeout, redisKey).Err()
}

func (s *Store) GetCounter(prefix string, keyId string) (int64, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	redisKey := prefix + "-" + keyId
	val := s.client.Get(ctxTimeout, redisKey)

	return val.Int64()
}
