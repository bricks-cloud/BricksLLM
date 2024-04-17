package user

import (
	"fmt"
	"strings"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/key"
)

type User struct {
	Id                     string           `json:"id"`
	Name                   string           `json:"name"`
	CreatedAt              int64            `json:"createdAt"`
	UpdatedAt              int64            `json:"updatedAt"`
	Tags                   []string         `json:"tags"`
	KeyIds                 []string         `json:"keyIds"`
	Revoked                bool             `json:"revoked"`
	RevokedReason          string           `json:"revokedReason"`
	CostLimitInUsd         float64          `json:"costLimitInUsd"`
	CostLimitInUsdOverTime float64          `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     key.TimeUnit     `json:"costLimitInUsdUnit"`
	RateLimitOverTime      int              `json:"rateLimitOverTime"`
	RateLimitUnit          key.TimeUnit     `json:"rateLimitUnit"`
	Ttl                    string           `json:"ttl"`
	AllowedPaths           []key.PathConfig `json:"allowedPaths"`
	AllowedModels          []string         `json:"allowedModels"`
	UserId                 string           `json:"userId"`
}

func (u *User) Validate() error {
	invalid := []string{}

	for _, tag := range u.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	for _, kid := range u.KeyIds {
		if len(kid) == 0 {
			invalid = append(invalid, "keyIds")
			break
		}
	}

	if u.UpdatedAt <= 0 {
		invalid = append(invalid, "updatedAt")
	}

	if u.CreatedAt <= 0 {
		invalid = append(invalid, "createdAt")
	}

	if len(u.Name) == 0 {
		invalid = append(invalid, "name")
	}

	if u.CostLimitInUsd < 0 {
		invalid = append(invalid, "costLimitInUsd")
	}

	if u.CostLimitInUsdOverTime < 0 {
		invalid = append(invalid, "costLimitInUsdOverTime")
	}

	if u.RateLimitOverTime < 0 {
		invalid = append(invalid, "rateLimitOverTime")
	}

	if len(u.Ttl) != 0 {
		_, err := time.ParseDuration(u.Ttl)
		if err != nil {
			invalid = append(invalid, "ttl")
		}
	}

	for index, p := range u.AllowedPaths {
		if len(p.Path) == 0 {
			invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.path", index))
			break
		}

		if len(p.Method) == 0 {
			invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.method", index))
			break
		}
	}

	for _, model := range u.AllowedModels {
		if len(model) == 0 {
			invalid = append(invalid, "allowedModels")
			break
		}
	}

	if len(invalid) > 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if len(u.RateLimitUnit) != 0 && u.RateLimitOverTime == 0 {
		return internal_errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
	}

	if len(u.CostLimitInUsdUnit) != 0 && u.CostLimitInUsdOverTime == 0 {
		return internal_errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
	}

	if u.RateLimitOverTime != 0 {
		if len(u.RateLimitUnit) == 0 {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if u.RateLimitUnit != key.HourTimeUnit && u.RateLimitUnit != key.MinuteTimeUnit && u.RateLimitUnit != key.SecondTimeUnit && u.RateLimitUnit != key.DayTimeUnit {
			return internal_errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if u.CostLimitInUsdOverTime != 0 {
		if len(u.CostLimitInUsdUnit) == 0 {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if u.CostLimitInUsdUnit != key.DayTimeUnit && u.CostLimitInUsdUnit != key.HourTimeUnit && u.CostLimitInUsdUnit != key.MonthTimeUnit && u.CostLimitInUsdUnit != key.MinuteTimeUnit {
			return internal_errors.NewValidationError("cost limit unit can not be identified")
		}
	}

	return nil
}

type UpdateUser struct {
	Name                   string           `json:"name"`
	UpdatedAt              int64            `json:"updatedAt"`
	Tags                   []string         `json:"tags"`
	KeyIds                 []string         `json:"keyIds"`
	Revoked                *bool            `json:"revoked"`
	RevokedReason          string           `json:"revokedReason"`
	CostLimitInUsd         *float64         `json:"costLimitInUsd"`
	CostLimitInUsdOverTime *float64         `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     *key.TimeUnit    `json:"costLimitInUsdUnit"`
	RateLimitOverTime      *int             `json:"rateLimitOverTime"`
	RateLimitUnit          *key.TimeUnit    `json:"rateLimitUnit"`
	AllowedPaths           []key.PathConfig `json:"allowedPaths"`
	AllowedModels          []string         `json:"allowedModels"`
	UserId                 string           `json:"userId"`
}

func (uu *UpdateUser) Validate() error {
	invalid := []string{}

	for _, tag := range uu.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	for _, kid := range uu.KeyIds {
		if len(kid) == 0 {
			invalid = append(invalid, "keyIds")
			break
		}
	}

	if uu.CostLimitInUsd != nil && *uu.CostLimitInUsd < 0 {
		invalid = append(invalid, "costLimitInUsd")
	}

	if uu.UpdatedAt <= 0 {
		invalid = append(invalid, "updatedAt")
	}

	for index, p := range uu.AllowedPaths {
		if len(p.Path) == 0 {
			invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.path", index))
			break
		}

		if len(p.Method) == 0 {
			invalid = append(invalid, fmt.Sprintf("allowedPaths.%d.method", index))
			break
		}
	}

	for _, model := range uu.AllowedModels {
		if len(model) == 0 {
			invalid = append(invalid, "allowedModels")
			break
		}
	}

	if len(invalid) > 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if uu.RateLimitUnit != nil {
		if uu.RateLimitOverTime == nil {
			return internal_errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
		}

		if len(*uu.RateLimitUnit) == 0 && *uu.RateLimitOverTime != 0 {
			return internal_errors.NewValidationError("rate limit over time must be 0 if rate limit unit is empty")
		}
	}

	if uu.CostLimitInUsdUnit != nil {
		if uu.CostLimitInUsdOverTime == nil {
			return internal_errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
		}

		if len(*uu.CostLimitInUsdUnit) == 0 && *uu.CostLimitInUsdOverTime != 0 {
			return internal_errors.NewValidationError("cost limit over time must be 0 if cost limit unit is empty")
		}
	}

	if uu.RateLimitOverTime != nil {
		if uu.RateLimitUnit == nil {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if *uu.RateLimitOverTime == 0 && len(*uu.RateLimitUnit) != 0 {
			return internal_errors.NewValidationError("rate limit unit has to be empty if rate limit over time is 0")
		}

		if *uu.RateLimitOverTime != 0 && len(*uu.RateLimitUnit) == 0 {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if *uu.RateLimitOverTime != 0 && *uu.RateLimitUnit != key.HourTimeUnit && *uu.RateLimitUnit != key.MinuteTimeUnit && *uu.RateLimitUnit != key.SecondTimeUnit && *uu.RateLimitUnit != key.DayTimeUnit {
			return internal_errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if uu.CostLimitInUsdOverTime != nil {
		if uu.CostLimitInUsdUnit == nil {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if *uu.CostLimitInUsdOverTime == 0 && len(*uu.CostLimitInUsdUnit) != 0 {
			return internal_errors.NewValidationError("cost limit unit has to be empty if cost limit over time is 0")
		}

		if *uu.CostLimitInUsdOverTime != 0 && len(*uu.CostLimitInUsdUnit) == 0 {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if *uu.CostLimitInUsdOverTime != 0 && *uu.CostLimitInUsdUnit != key.DayTimeUnit && *uu.CostLimitInUsdUnit != key.HourTimeUnit && *uu.CostLimitInUsdUnit != key.MonthTimeUnit && *uu.CostLimitInUsdUnit != key.MinuteTimeUnit {
			return internal_errors.NewValidationError("cost limit unit can not be identified")
		}
	}

	return nil
}
