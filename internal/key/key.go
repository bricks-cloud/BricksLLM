package key

import (
	"fmt"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/errors"
)

type Provider string

const (
	OpenAiProvider Provider = "openai"
)

type UpdateKey struct {
	Name                   string        `json:"name"`
	UpdatedAt              int64         `json:"updatedAt"`
	Tags                   []string      `json:"tags"`
	Revoked                *bool         `json:"revoked"`
	Retrievable            *bool         `json:"retrievable"`
	CostLimitInUsd         float64       `json:"costLimitInUsd"`
	CostLimitInUsdOverTime float64       `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     TimeUnit      `json:"costLimitInUsdUnit"`
	RateLimitOverTime      int           `json:"rateLimitOverTime"`
	RateLimitUnit          TimeUnit      `json:"rateLimitUnit"`
	Ttl                    time.Duration `json:"ttl"`
}

func (uk *UpdateKey) Validate() error {
	invalid := []string{}

	for _, tag := range uk.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	if uk.UpdatedAt <= 0 {
		invalid = append(invalid, "updatedAt")
	}

	if uk.CostLimitInUsd < 0 {
		invalid = append(invalid, "costLimitInUsd")
	}

	if uk.CostLimitInUsdOverTime < 0 {
		invalid = append(invalid, "costLimitInUsdOverTime")
	}

	if uk.RateLimitOverTime < 0 {
		invalid = append(invalid, "rateLimitOverTime")
	}

	if uk.Ttl < 0 {
		invalid = append(invalid, "ttl")
	}

	if len(invalid) > 0 {
		return errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if len(uk.RateLimitUnit) != 0 && uk.RateLimitOverTime == 0 {
		return errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
	}

	if len(uk.CostLimitInUsdUnit) != 0 && uk.CostLimitInUsdOverTime == 0 {
		return errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
	}

	if uk.RateLimitOverTime != 0 {
		if len(uk.RateLimitUnit) == 0 {
			return errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if uk.RateLimitUnit != HourTimeUnit && uk.RateLimitUnit != MinuteTimeUnit && uk.RateLimitUnit != SecondTimeUnit && uk.RateLimitUnit != DayTimeUnit {
			return errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if uk.CostLimitInUsdOverTime != 0 {
		if len(uk.CostLimitInUsdUnit) == 0 {
			return errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if uk.CostLimitInUsdUnit != HourTimeUnit && uk.CostLimitInUsdUnit != DayTimeUnit {
			return errors.NewValidationError("cost limit unit can not be identified")
		}
	}

	return nil
}

type RequestKey struct {
	Name                   string        `json:"name"`
	CreatedAt              int64         `json:"createdAt"`
	UpdatedAt              int64         `json:"updatedAt"`
	Tags                   []string      `json:"tags"`
	Revoked                *bool         `json:"revoked"`
	KeyId                  string        `json:"keyId"`
	Key                    string        `json:"key"`
	Retrievable            *bool         `json:"retrievable"`
	CostLimitInUsd         float64       `json:"costLimitInUsd"`
	CostLimitInUsdOverTime float64       `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     TimeUnit      `json:"costLimitInUsdUnit"`
	RateLimitOverTime      int           `json:"rateLimitOverTime"`
	RateLimitUnit          TimeUnit      `json:"rateLimitUnit"`
	Ttl                    time.Duration `json:"ttl"`
}

func (rk *RequestKey) Validate() error {
	invalid := []string{}

	for _, tag := range rk.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	if rk.UpdatedAt <= 0 {
		invalid = append(invalid, "updatedAt")
	}

	if rk.Revoked == nil {
		invalid = append(invalid, "revoked")
	}

	if rk.Retrievable == nil {
		invalid = append(invalid, "retrievable")
	}

	if len(rk.Key) == 0 {
		invalid = append(invalid, "key")
	}

	if rk.CreatedAt <= 0 {
		invalid = append(invalid, "createdAt")
	}

	if len(rk.Name) == 0 {
		invalid = append(invalid, "name")
	}

	if len(rk.KeyId) == 0 {
		invalid = append(invalid, "keyId")
	}

	if rk.CostLimitInUsd < 0 {
		invalid = append(invalid, "costLimitInUsd")
	}

	if rk.CostLimitInUsdOverTime < 0 {
		invalid = append(invalid, "costLimitInUsdOverTime")
	}

	if rk.RateLimitOverTime < 0 {
		invalid = append(invalid, "rateLimitOverTime")
	}

	if rk.Ttl < 0 {
		invalid = append(invalid, "ttl")
	}

	if len(invalid) > 0 {
		return errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if len(rk.RateLimitUnit) != 0 && rk.RateLimitOverTime == 0 {
		return errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
	}

	if len(rk.CostLimitInUsdUnit) != 0 && rk.CostLimitInUsdOverTime == 0 {
		return errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
	}

	if rk.RateLimitOverTime != 0 {
		if len(rk.RateLimitUnit) == 0 {
			return errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if rk.RateLimitUnit != HourTimeUnit && rk.RateLimitUnit != MinuteTimeUnit && rk.RateLimitUnit != SecondTimeUnit && rk.RateLimitUnit != DayTimeUnit {
			return errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if rk.CostLimitInUsdOverTime != 0 {
		if len(rk.CostLimitInUsdUnit) == 0 {
			return errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if rk.CostLimitInUsdUnit != DayTimeUnit && rk.CostLimitInUsdUnit != HourTimeUnit {
			return errors.NewValidationError("cost limit unit can not be identified")
		}
	}

	return nil
}

type TimeUnit string

const (
	HourTimeUnit   TimeUnit = "h"
	MinuteTimeUnit TimeUnit = "m"
	SecondTimeUnit TimeUnit = "s"
	DayTimeUnit    TimeUnit = "d"
)

type ResponseKey struct {
	Name                   string        `json:"name"`
	CreatedAt              int64         `json:"createdAt"`
	UpdatedAt              int64         `json:"updatedAt"`
	Tags                   []string      `json:"tags"`
	KeyId                  string        `json:"keyId"`
	Revoked                bool          `json:"revoked"`
	Key                    string        `json:"key"`
	Retrievable            bool          `json:"retrievable"`
	CostLimitInUsd         float64       `json:"costLimitInUsd"`
	CostLimitInUsdOverTime float64       `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     TimeUnit      `json:"costLimitInUsdUnit"`
	RateLimitOverTime      int           `json:"rateLimitOverTime"`
	RateLimitUnit          TimeUnit      `json:"rateLimitUnit"`
	Ttl                    time.Duration `json:"ttl"`
}
