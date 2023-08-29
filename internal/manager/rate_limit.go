package manager

import "github.com/bricks-cloud/bricksllm/internal/key"

type Cache interface {
	IncrementCounter(keyId string, rateLimitUnit key.TimeUnit, incr int64) error
}

type RateLimitManager struct {
	c Cache
}

func NewRateLimitManager(c Cache) *RateLimitManager {
	return &RateLimitManager{
		c: c,
	}
}

func (rlm *RateLimitManager) Increment(keyId string, timeUnit key.TimeUnit) error {
	err := rlm.c.IncrementCounter(keyId, timeUnit, 1)

	if err != nil {
		return err
	}

	return nil
}
