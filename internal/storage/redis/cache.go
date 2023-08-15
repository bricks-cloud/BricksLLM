package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	wt     time.Duration
	rt     time.Duration
}

func NewCache(c *redis.Client, wt time.Duration, rt time.Duration) *Store {
	return &Store{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (c *Cache) IncrementCounter(prefix string, keyId string, rateLimitUnit key.TimeUnit, incr int64) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), c.wt)
	defer cancel()

	redisKey := prefix + "-" + keyId
	ts, err := getCounterTimeStamp(rateLimitUnit)
	if err != nil {
		return err
	}

	err = c.client.HIncrBy(ctxTimeout, redisKey, strconv.FormatInt(ts, 10), incr).Err()
	if err != nil {
		return err
	}

	ctxTimeout, cancel = context.WithTimeout(context.Background(), c.rt)
	defer cancel()
	dur := c.client.TTL(ctxTimeout, redisKey)
	err = dur.Err()
	if err != nil {
		return err
	}

	if dur.Val() == -2 {
		ttl, err := getCounterTtl(rateLimitUnit)
		if err != nil {
			return err
		}

		ctxTimeout, cancel = context.WithTimeout(context.Background(), c.wt)
		defer cancel()
		err = c.client.Expire(ctxTimeout, redisKey, ttl).Err()
		if err != nil {
			return err
		}

	}

	return nil
}

func getCounterTtl(rateLimitUnit key.TimeUnit) (time.Duration, error) {
	switch rateLimitUnit {
	case key.SecondTimeUnit:
		return time.Second, nil
	case key.HourTimeUnit:
		return time.Hour, nil
	case key.DayTimeUnit:
		return time.Hour * 24, nil
	case key.MinuteTimeUnit:
		return time.Minute, nil
	}

	return 0, fmt.Errorf("cannot recognize rate limit time unit %v", rateLimitUnit)
}

func getCounterTimeStamp(rateLimitUnit key.TimeUnit) (int64, error) {
	now := time.Now().UTC()
	switch rateLimitUnit {
	case key.SecondTimeUnit:
		return now.UnixMilli() * 10, nil
	case key.HourTimeUnit:
		return int64(now.Minute()), nil
	case key.DayTimeUnit:
		return int64(now.Hour()), nil
	case key.MinuteTimeUnit:
		return now.Unix(), nil
	}

	return 0, fmt.Errorf("cannot recognize rate limit time unit %v", rateLimitUnit)
}

func (c *Cache) GetCounter(prefix string, keyId string, rateLimitUnit key.TimeUnit) (int64, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), c.rt)
	defer cancel()

	redisKey := prefix + "-" + keyId
	strSlices := c.client.HVals(ctxTimeout, redisKey)
	err := strSlices.Err()
	if err != nil {
		return 0, err
	}

	strVals := strSlices.Val()
	intVals := []int64{}

	for _, strVal := range strVals {
		intVal, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return 0, nil
		}

		intVals = append(intVals, intVal)
	}

	var counter int64 = 0
	for _, intVal := range intVals {
		counter += intVal
	}

	return counter, nil

}
