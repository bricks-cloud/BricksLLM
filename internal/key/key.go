package key

import (
	"errors"
	"fmt"
	"strings"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
)

const RevokedReasonExpired string = "expired"

type UpdateKey struct {
	Name                   string        `json:"name"`
	UpdatedAt              int64         `json:"updatedAt"`
	Tags                   []string      `json:"tags"`
	Revoked                *bool         `json:"revoked"`
	RevokedReason          string        `json:"revokedReason"`
	Key                    string        `json:"key"`
	SettingId              string        `json:"settingId"`
	SettingIds             []string      `json:"settingIds"`
	CostLimitInUsd         *float64      `json:"costLimitInUsd"`
	CostLimitInUsdOverTime *float64      `json:"costLimitInUsdOverTime"`
	CostLimitInUsdUnit     *TimeUnit     `json:"costLimitInUsdUnit"`
	RateLimitOverTime      *int          `json:"rateLimitOverTime"`
	RateLimitUnit          *TimeUnit     `json:"rateLimitUnit"`
	AllowedPaths           *[]PathConfig `json:"allowedPaths,omitempty"`
	ShouldLogRequest       *bool         `json:"shouldLogRequest"`
	ShouldLogResponse      *bool         `json:"shouldLogResponse"`
	RotationEnabled        *bool         `json:"rotationEnabled"`
	PolicyId               *string       `json:"policyId"`
	IsKeyNotHashed         *bool         `json:"isKeyNotHashed"`
}

func (uk *UpdateKey) Validate() error {
	invalid := []string{}

	if len(uk.SettingId) != 0 && len(uk.SettingIds) != 0 {
		return errors.New("either settingId or settingIds")
	}

	for _, tag := range uk.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	if uk.CostLimitInUsd != nil && *uk.CostLimitInUsd < 0 {
		invalid = append(invalid, "costLimitInUsd")
	}

	if uk.UpdatedAt <= 0 {
		invalid = append(invalid, "updatedAt")
	}

	if len(uk.SettingIds) != 0 {
		for index, id := range uk.SettingIds {
			if len(id) == 0 {
				invalid = append(invalid, fmt.Sprintf("settingIds[%d]", index))
			}
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

	if uk.PolicyId != nil {
		if len(*uk.PolicyId) == 0 {
			invalid = append(invalid, "policyId")
		}
	}

	if len(invalid) > 0 {
		return internal_errors.NewValidationError(fmt.Sprintf("fields [%s] are invalid", strings.Join(invalid, ", ")))
	}

	if uk.RateLimitUnit != nil {
		if uk.RateLimitOverTime == nil {
			return internal_errors.NewValidationError("rate limit over time can not be empty if rate limit unit is specified")
		}

		if len(*uk.RateLimitUnit) == 0 && *uk.RateLimitOverTime != 0 {
			return internal_errors.NewValidationError("rate limit over time must be 0 if rate limit unit is empty")
		}
	}

	if uk.CostLimitInUsdUnit != nil {
		if uk.CostLimitInUsdOverTime == nil {
			return internal_errors.NewValidationError("cost limit over time can not be empty if cost limit unit is specified")
		}

		if len(*uk.CostLimitInUsdUnit) == 0 && *uk.CostLimitInUsdOverTime != 0 {
			return internal_errors.NewValidationError("cost limit over time must be 0 if cost limit unit is empty")
		}
	}

	if uk.RateLimitOverTime != nil {
		if uk.RateLimitUnit == nil {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if *uk.RateLimitOverTime == 0 && len(*uk.RateLimitUnit) != 0 {
			return internal_errors.NewValidationError("rate limit unit has to be empty if rate limit over time is 0")
		}

		if *uk.RateLimitOverTime != 0 && len(*uk.RateLimitUnit) == 0 {
			return internal_errors.NewValidationError("rate limit unit can not be empty if rate limit over time is specified")
		}

		if *uk.RateLimitOverTime != 0 && *uk.RateLimitUnit != HourTimeUnit && *uk.RateLimitUnit != MinuteTimeUnit && *uk.RateLimitUnit != SecondTimeUnit && *uk.RateLimitUnit != DayTimeUnit {
			return internal_errors.NewValidationError("rate limit unit can not be identified")
		}
	}

	if uk.CostLimitInUsdOverTime != nil {
		if uk.CostLimitInUsdUnit == nil {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if *uk.CostLimitInUsdOverTime == 0 && len(*uk.CostLimitInUsdUnit) != 0 {
			return internal_errors.NewValidationError("cost limit unit has to be empty if cost limit over time is 0")
		}

		if *uk.CostLimitInUsdOverTime != 0 && len(*uk.CostLimitInUsdUnit) == 0 {
			return internal_errors.NewValidationError("cost limit unit can not be empty if cost limit over time is specified")
		}

		if *uk.CostLimitInUsdOverTime != 0 && *uk.CostLimitInUsdUnit != DayTimeUnit && *uk.CostLimitInUsdUnit != HourTimeUnit && *uk.CostLimitInUsdUnit != MonthTimeUnit && *uk.CostLimitInUsdUnit != MinuteTimeUnit {
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
	SettingIds             []string     `json:"settingIds"`
	ShouldLogRequest       bool         `json:"shouldLogRequest"`
	ShouldLogResponse      bool         `json:"shouldLogResponse"`
	RotationEnabled        bool         `json:"rotationEnabled"`
	PolicyId               string       `json:"policyId"`
	IsKeyNotHashed         bool         `json:"isKeyNotHashed"`
}

func (rk *RequestKey) Validate() error {
	invalid := []string{}

	if len(rk.SettingId) == 0 && len(rk.SettingIds) == 0 {
		return errors.New("settingId is not set in either setting_id or setting_ids field")
	}

	if len(rk.SettingId) != 0 && len(rk.SettingIds) != 0 {
		return errors.New("either settingId or settingIds")
	}

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

	if len(rk.SettingIds) != 0 {
		for index, id := range rk.SettingIds {
			if len(id) == 0 {
				invalid = append(invalid, fmt.Sprintf("settingIds[%d]", index))
			}
		}
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

		if rk.CostLimitInUsdUnit != DayTimeUnit && rk.CostLimitInUsdUnit != HourTimeUnit && rk.CostLimitInUsdUnit != MonthTimeUnit && rk.CostLimitInUsdUnit != MinuteTimeUnit {
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
	MonthTimeUnit  TimeUnit = "mo"
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
	SettingIds             []string     `json:"settingIds"`
	ShouldLogRequest       bool         `json:"shouldLogRequest"`
	ShouldLogResponse      bool         `json:"shouldLogResponse"`
	RotationEnabled        bool         `json:"rotationEnabled"`
	PolicyId               string       `json:"policyId"`
	IsKeyNotHashed         bool         `json:"isKeyNotHashed"`
}

func (rk *ResponseKey) GetSettingIds() []string {
	settingIds := []string{}
	if len(rk.SettingId) != 0 {
		settingIds = append(settingIds, rk.SettingId)
	}

	for _, settingId := range rk.SettingIds {
		if len(settingId) != 0 {
			settingIds = append(settingIds, settingId)
		}
	}

	return settingIds
}
