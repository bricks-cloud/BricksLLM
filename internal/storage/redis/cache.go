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

func NewCache(c *redis.Client, wt time.Duration, rt time.Duration) *Cache {
	return &Cache{
		client: c,
		wt:     wt,
		rt:     rt,
	}
}

func (c *Cache) IncrementCounter(keyId string, timeUnit key.TimeUnit, incr int64) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), c.wt)
	defer cancel()

	ts, err := getCounterTimeStamp(timeUnit)
	if err != nil {
		return err
	}

	err = c.client.HIncrBy(ctxTimeout, keyId, strconv.FormatInt(ts, 10), incr).Err()
	if err != nil {
		return err
	}

	ctxTimeout, cancel = context.WithTimeout(context.Background(), c.rt)
	defer cancel()
	dur := c.client.TTL(ctxTimeout, keyId)
	err = dur.Err()
	if err != nil {
		return err
	}

	val := dur.Val()
	if val < 0 {
		ttl, err := getCounterTtl(timeUnit)
		if err != nil {
			return err
		}

		ctxTimeout, cancel = context.WithTimeout(context.Background(), c.wt)
		defer cancel()
		err = c.client.ExpireAt(ctxTimeout, keyId, ttl).Err()
		if err != nil {
			return err
		}

	}

	return nil
}

func getCounterTtl(rateLimitUnit key.TimeUnit) (time.Time, error) {
	now := time.Now().UTC()
	switch rateLimitUnit {
	case key.SecondTimeUnit:
		return now.Truncate(time.Second).Add(time.Second).Add(-time.Millisecond), nil
	case key.MinuteTimeUnit:
		return now.Truncate(60 * time.Second).Add(time.Second * 60).Add(-time.Millisecond), nil
	case key.HourTimeUnit:
		return now.Truncate(60 * time.Minute).Add(time.Minute * 60).Add(-time.Millisecond), nil
	case key.DayTimeUnit:
		return now.Truncate(24 * time.Hour).Add(time.Hour * 24).Add(-time.Millisecond), nil
	case key.MonthTimeUnit:
		firstDayOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		return firstDayOfNextMonth.Add(-time.Millisecond), nil
	}

	return time.Time{}, fmt.Errorf("cannot recognize rate limit time unit %v", rateLimitUnit)
}

func getCounterTimeStamp(rateLimitUnit key.TimeUnit) (int64, error) {
	now := time.Now().UTC()
	switch rateLimitUnit {
	case key.SecondTimeUnit:
		return now.UnixMilli() * 10, nil
	case key.MinuteTimeUnit:
		return now.Unix(), nil
	case key.HourTimeUnit:
		return int64(now.Minute()), nil
	case key.DayTimeUnit:
		return int64(now.Hour()), nil
	case key.MonthTimeUnit:
		return int64(now.Day()), nil
	}

	return 0, fmt.Errorf("cannot recognize rate limit time unit %v", rateLimitUnit)
}

func (c *Cache) GetCounter(keyId string, rateLimitUnit key.TimeUnit) (int64, error) {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), c.rt)
	defer cancel()

	strSlices := c.client.HVals(ctxTimeout, keyId)
	err := strSlices.Err()

	if err != nil && err != redis.Nil {
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
