package key

import (
	"fmt"
	"strings"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
)

type UpdateKey struct {
	Name                   string        `json:"name"`
	UpdatedAt              int64         `json:"updatedAt"`
	Tags                   []string      `json:"tags"`
	Revoked                *bool         `json:"revoked"`
	RevokedReason          string        `json:"revokedReason"`
	SettingId              string        `json:"settingId"`
	AllowedPaths           *[]PathConfig `json:"allowedPaths,omitempty"`
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

	if len(uk.Ttl) != 0 {
		_, err := time.ParseDuration(uk.Ttl)
		if err != nil {
			invalid = append(invalid, "ttl")
		}
	}

	if uk.AllowedPaths != nil {
		for index, p := range *uk.AllowedPaths {
			if len(p.Path) == 0 {
				invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.path", index))
				break
			}

			if len(p.Method) == 0 {
				invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.method", index))
				break
			}
		}
	}

	if len(invalid) > 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if len(uk.RateLimitUnit) != 0 && uk.RateLimitOverTime == 0 {
		return internal_errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
	}

	if len(uk.CostLimitInUsdUnit) != 0 && uk.CostLimitInUsdOverTime == 0 {
		return internal_errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
	}

	if uk.RateLimitOverTime != 0 {
		if len(uk.RateLimitUnit) == 0 {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if uk.RateLimitUnit != HourTimeUnit && uk.RateLimitUnit != MinuteTimeUnit && uk.RateLimitUnit != SecondTimeUnit && uk.RateLimitUnit != DayTimeUnit {
			return internal_errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if uk.CostLimitInUsdOverTime != 0 {
		if len(uk.CostLimitInUsdUnit) == 0 {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if uk.CostLimitInUsdUnit != HourTimeUnit && uk.CostLimitInUsdUnit != DayTimeUnit {
			return internal_errors.NewValidationError("cost limit unit can not be identified")
		}
	}

	return nil
}

type PathConfig struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type RequestKey struct {
	Name                   string       `json:"name"`
	CreatedAt              int64        `json:"createdAt"`
	UpdatedAt              int64        `json:"updatedAt"`
	Tags                   []string     `json:"tags"`
	KeyId                  string       `json:"keyId"`
	Key                    string       `json:"key"`
	CostLimitInUsd         float64      `json:"costLimitInUsd"`
	CostLimitInUsdOverTime float64      `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     TimeUnit     `json:"costLimitInUsdUnit"`
	RateLimitOverTime      int          `json:"rateLimitOverTime"`
	RateLimitUnit          TimeUnit     `json:"rateLimitUnit"`
	Ttl                    string       `json:"ttl"`
	SettingId              string       `json:"settingId"`
	AllowedPaths           []PathConfig `json:"allowedPaths"`
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

	if len(rk.SettingId) == 0 {
		invalid = append(invalid, "settingId")
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

	if len(rk.Ttl) != 0 {
		_, err := time.ParseDuration(rk.Ttl)
		if err != nil {
			invalid = append(invalid, "ttl")
		}
	}

	if len(rk.AllowedPaths) != 0 {
		for index, p := range rk.AllowedPaths {
			if len(p.Path) == 0 {
				invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.path", index))
				break
			}

			if len(p.Method) == 0 {
				invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.method", index))
				break
			}
		}
	}

	if len(invalid) > 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if len(rk.RateLimitUnit) != 0 && rk.RateLimitOverTime == 0 {
		return internal_errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
	}

	if len(rk.CostLimitInUsdUnit) != 0 && rk.CostLimitInUsdOverTime == 0 {
		return internal_errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
	}

	if rk.RateLimitOverTime != 0 {
		if len(rk.RateLimitUnit) == 0 {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if rk.RateLimitUnit != HourTimeUnit && rk.RateLimitUnit != MinuteTimeUnit && rk.RateLimitUnit != SecondTimeUnit && rk.RateLimitUnit != DayTimeUnit {
			return internal_errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if rk.CostLimitInUsdOverTime != 0 {
		if len(rk.CostLimitInUsdUnit) == 0 {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if rk.CostLimitInUsdUnit != DayTimeUnit && rk.CostLimitInUsdUnit != HourTimeUnit {
			return internal_errors.NewValidationError("cost limit unit can not be identified")
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
	Name                   string       `json:"name"`
	CreatedAt              int64        `json:"createdAt"`
	UpdatedAt              int64        `json:"updatedAt"`
	Tags                   []string     `json:"tags"`
	KeyId                  string       `json:"keyId"`
	Revoked                bool         `json:"revoked"`
	Key                    string       `json:"key"`
	RevokedReason          string       `json:"revokedReason"`
	CostLimitInUsd         float64      `json:"costLimitInUsd"`
	CostLimitInUsdOverTime float64      `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     TimeUnit     `json:"costLimitInUsdUnit"`
	RateLimitOverTime      int          `json:"rateLimitOverTime"`
	RateLimitUnit          TimeUnit     `json:"rateLimitUnit"`
	Ttl                    string       `json:"ttl"`
	SettingId              string       `json:"settingId"`
	AllowedPaths           []PathConfig `json:"allowedPaths"`
}
