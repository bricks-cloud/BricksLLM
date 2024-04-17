package validator

import (
	"errors"
	"fmt"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/user"
)

type UserValidator struct {
	clc costLimitCache
	rlc rateLimitCache
	cls costLimitStorage
}

func NewUserValidator(
	clc costLimitCache,
	rlc rateLimitCache,
	cls costLimitStorage,
) *UserValidator {
	return &UserValidator{
		clc: clc,
		rlc: rlc,
		cls: cls,
	}
}

func (v *UserValidator) Validate(u *user.User, promptCost float64) error {
	if u == nil {
		return internal_errors.NewValidationError("empty user")
	}

	if u.Revoked {
		return internal_errors.NewValidationError("user revoked")
	}

	parsed, err := time.ParseDuration(u.Ttl)
	if err != nil {
		return err
	}

	if !v.validateTtl(u.CreatedAt, parsed) {
		return internal_errors.NewExpirationError("user expired", internal_errors.TtlExpiration)
	}

	err = v.validateRateLimitOverTime(u.UserId, u.RateLimitOverTime, u.RateLimitUnit)
	if err != nil {
		return err
	}

	err = v.validateCostLimitOverTime(u.UserId, u.CostLimitInUsdOverTime, u.CostLimitInUsdUnit)
	if err != nil {
		return err
	}

	err = v.validateCostLimit(u.UserId, u.CostLimitInUsd)
	if err != nil {
		return err
	}

	return nil
}

func (v *UserValidator) validateTtl(createdAt int64, ttl time.Duration) bool {
	ttlInSecs := int64(ttl.Seconds())

	if ttlInSecs == 0 {
		return true
	}

	current := time.Now().Unix()
	return current < createdAt+ttlInSecs
}

func (v *UserValidator) validateRateLimitOverTime(userId string, rateLimitOverTime int, rateLimitUnit key.TimeUnit) error {
	if rateLimitOverTime == 0 {
		return nil
	}

	c, err := v.rlc.GetCounter(userId, rateLimitUnit)
	if err != nil {
		return errors.New("failed to get rate limit counter")
	}

	if c >= int64(rateLimitOverTime) {
		return internal_errors.NewRateLimitError(fmt.Sprintf("user exceeded rate limit %d requests per %s", rateLimitOverTime, rateLimitUnit))
	}

	return nil
}

func (v *UserValidator) validateCostLimitOverTime(userId string, costLimitOverTime float64, costLimitUnit key.TimeUnit) error {
	if costLimitOverTime == 0 {
		return nil
	}

	cachedCost, err := v.clc.GetCounter(userId, costLimitUnit)
	if err != nil {
		return errors.New("failed to get cached token cost")
	}

	if cachedCost >= convertDollarToMicroDollars(costLimitOverTime) {
		return internal_errors.NewCostLimitError(fmt.Sprintf("cost limit: %f has been reached for the current time period: %s", costLimitOverTime, costLimitUnit))
	}

	return nil
}

func (v *UserValidator) validateCostLimit(userId string, costLimit float64) error {
	if costLimit == 0 {
		return nil
	}

	existingTotalCost, err := v.cls.GetCounter(userId)
	if err != nil {
		return errors.New("failed to get total token cost")
	}

	if existingTotalCost >= convertDollarToMicroDollars(costLimit) {
		return internal_errors.NewExpirationError(fmt.Sprintf("total cost limit: %f has been reached", costLimit), internal_errors.CostLimitExpiration)
	}

	return nil
}
