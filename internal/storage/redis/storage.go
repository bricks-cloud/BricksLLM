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

func (s *Store) IncrementCounter(keyId string, incr int64) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	return s.client.IncrBy(ctxTimeout, keyId, incr).Err()
}

func (s *Store) DeleteCounter(keyId string) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.wt)
	defer cancel()

	return s.client.Del(ctxTimeout, keyId).Err()
}

func (s *Store) GetCounter(keyId string) (int64, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), s.rt)
	defer cancel()

	val := s.client.Get(ctxTimeout, keyId)
	result, err := val.Int64()
	if err == nil {
		return result, nil
	}

	if err == redis.Nil {
		return 0, nil
	}

	return 0, err
}
