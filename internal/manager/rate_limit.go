package manager

import "github.com/bricks-cloud/bricksllm/internal/key"

type Cache interface {
	IncrementCounter(keyId string, rateLimitUnit key.TimeUnit, incr int64) error
}

type RateLimitManager struct {
	c  Cache
	uc Cache
}

func NewRateLimitManager(c Cache, uc Cache) *RateLimitManager {
	return &RateLimitManager{
		c:  c,
		uc: uc,
	}
}

func (rlm *RateLimitManager) Increment(keyId string, timeUnit key.TimeUnit) error {
	err := rlm.c.IncrementCounter(keyId, timeUnit, 1)

	if err != nil {
		return err
	}

	return nil
}

func (rlm *RateLimitManager) IncrementUser(id string, timeUnit key.TimeUnit) error {
	err := rlm.uc.IncrementCounter(id, timeUnit, 1)

	if err != nil {
		return err
	}

	return nil
}
