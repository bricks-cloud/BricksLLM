package validator

import (
	"errors"
	"fmt"
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/key"
)

const (
	openAiCostPrefix      string = "openai-cost"
	openAiTotalCostPrefix string = "openai-total-cost"
	rateLimitPrefix       string = "rate-limit"
)

type counterCache interface {
	GetCounter(prefix string, keyId string, rateLimitUnit key.TimeUnit) (int64, error)
}

type storageCounterCache interface {
	GetCounter(prefix string, keyId string) (int64, error)
}

type Validator struct {
	cc                    counterCache
	scc                   storageCounterCache
	openAiCostPrefix      string
	openAiTotalCostPrefix string
	rateLimitPrefix       string
}

func NewValidator(
	cc counterCache,
	scc storageCounterCache,
	openAiCostPrefix string,
	openAiTotalCostPrefix string,
	rateLimitPrefix string,
) *Validator {
	return &Validator{
		cc:                    cc,
		scc:                   scc,
		openAiCostPrefix:      openAiCostPrefix,
		openAiTotalCostPrefix: openAiTotalCostPrefix,
		rateLimitPrefix:       rateLimitPrefix,
	}
}

func (v *Validator) Validate(k *key.ResponseKey, promptCost float64, model string) error {
	if k == nil {
		return internal_errors.NewValidationError("empty api key")
	}

	if k.Revoked {
		return internal_errors.NewValidationError("api key revoked")
	}

	parsed, err := time.ParseDuration(k.Ttl)
	if !v.validateTtl(k.CreatedAt, parsed) {
		return internal_errors.NewExpirationError("api key expired", internal_errors.TtlExpiration)
	}

	err = v.validateRateLimitOverTime(k.KeyId, k.RateLimitOverTime, k.RateLimitUnit)
	if err != nil {
		return err
	}

	err = v.validateCostLimitOverTime(k.KeyId, k.CostLimitInUsdOverTime, k.CostLimitInUsdUnit, promptCost)
	if err != nil {
		return err
	}

	err = v.validateCostLimit(k.KeyId, k.CostLimitInUsd, promptCost)
	if err != nil {
		return err
	}

	return nil
}

func (v *Validator) validateTtl(createdAt int64, ttl time.Duration) bool {
	ttlInSecs := int64(ttl.Seconds())

	if ttlInSecs == 0 {
		return true
	}

	current := time.Now().Unix()
	if current > createdAt+ttlInSecs {
		return false
	}

	return true
}

func (v *Validator) validateRateLimitOverTime(keyId string, rateLimitOverTime int, rateLimitUnit key.TimeUnit) error {
	if rateLimitOverTime == 0 {
		return nil
	}

	c, err := v.cc.GetCounter(v.rateLimitPrefix, keyId, rateLimitUnit)
	if err != nil {
		return errors.New("failed to get rate limit counter")
	}

	if c > int64(rateLimitOverTime) {
		return internal_errors.NewRateLimitError(fmt.Sprintf("key exceeded rate limit %d requests per %s", rateLimitOverTime, rateLimitUnit))
	}

	return nil
}

func (v *Validator) validateCostLimitOverTime(keyId string, costLimitOverTime float64, costLimitUnit key.TimeUnit, promptCost float64) error {
	if costLimitOverTime == 0 {
		return nil
	}

	existingCostInMicroDollars, err := v.cc.GetCounter(v.openAiCostPrefix, keyId, costLimitUnit)
	if err != nil {
		return errors.New("failed to get cached token cost")
	}

	if convertDollarToMicroDollars(promptCost)+existingCostInMicroDollars > convertDollarToMicroDollars(costLimitOverTime) {
		return internal_errors.NewRateLimitError(fmt.Sprintf("cost limit: %f has been reached for the current time period: %s", costLimitOverTime, costLimitUnit))
	}

	return nil
}

func convertDollarToMicroDollars(dollar float64) int64 {
	return int64(dollar * 1000000)
}

func (v *Validator) validateCostLimit(keyId string, costLimit float64, promptCost float64) error {
	if costLimit == 0 {
		return nil
	}

	existingTotalCost, err := v.scc.GetCounter(v.openAiTotalCostPrefix, keyId)
	if err != nil {
		return errors.New("failed to get total token cost")
	}

	if convertDollarToMicroDollars(promptCost)+existingTotalCost > convertDollarToMicroDollars(costLimit) {
		return internal_errors.NewExpirationError(fmt.Sprintf("total cost limit: %f has been reached", costLimit), internal_errors.CostLimitExpiration)
	}

	return nil
}
