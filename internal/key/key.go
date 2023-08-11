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

type RequestKey struct {
	Name                 string        `json:"name"`
	CreatedAt            int64         `json:"createdAt"`
	UpdatedAt            int64         `json:"updatedAt"`
	Tags                 []string      `json:"tags"`
	Revoked              *bool         `json:"revoked"`
	KeyId                string        `json:"keyId"`
	Key                  string        `json:"key"`
	Retrievable          *bool         `json:"retrievable"`
	CostLimitInUsd       float64       `json:"costLimitInUsd"`
	CostLimitInUsdPerDay float64       `json:"costLimitInUsdPerDay"`
	RateLimit            int           `json:"rateLimit"`
	RateLimitDuration    time.Duration `json:"rateLimitDuration"`
	Ttl                  time.Duration `json:"ttl"`
}

func (rk *RequestKey) Validate() error {
	invalid := []string{}
	if len(rk.Name) == 0 {
		invalid = append(invalid, "name")
	}

	if rk.CreatedAt <= 0 {
		invalid = append(invalid, "createdAt")
	}

	if rk.UpdatedAt <= 0 {
		invalid = append(invalid, "updatedAt")
	}

	for _, tag := range rk.Tags {
		if len(tag) == 0 {
			invalid = append(invalid, "tags")
			break
		}
	}

	if len(rk.KeyId) == 0 {
		invalid = append(invalid, "keyId")
	}

	if rk.CostLimitInUsd <= 0 {
		invalid = append(invalid, "costLimitInUsd")
	}

	if rk.CostLimitInUsdPerDay <= 0 {
		invalid = append(invalid, "costLimitInUsdPerDay")
	}

	if rk.RateLimit <= 0 {
		invalid = append(invalid, "rateLimit")
	}

	if rk.RateLimitDuration <= 0 {
		invalid = append(invalid, "rateLimitDuration")
	}

	if len(invalid) > 0 {
		return errors.NewValidationError(fmt.Sprintf("these fields: [%s] are invalid", strings.Join(invalid, ",")))
	}

	return nil
}

type ResponseKey struct {
	Name                 string        `json:"name"`
	CreatedAt            int64         `json:"createdAt"`
	UpdatedAt            int64         `json:"updatedAt"`
	Tags                 []string      `json:"tags"`
	KeyId                string        `json:"keyId"`
	Revoked              bool          `json:"revoked"`
	Key                  string        `json:"key"`
	Retrievable          bool          `json:"retrievable"`
	CostLimitInUsd       float64       `json:"costLimitInUsd"`
	CostLimitInUsdPerDay float64       `json:"costLimitInUsdPerDay"`
	RateLimit            int           `json:"rateLimit"`
	RateLimitDuration    time.Duration `json:"rateLimitDuration"`
	Ttl                  time.Duration `json:"ttl"`
}
