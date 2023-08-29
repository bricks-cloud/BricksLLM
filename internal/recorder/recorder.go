package recorder

import (
	"github.com/bricks-cloud/bricksllm/internal/key"
)

type Recorder struct {
	s  Store
	c  Cache
	ce CostEstimator
}

type Store interface {
	IncrementCounter(keyId string, incr int64) error
}

type Cache interface {
	IncrementCounter(keyId string, rateLimitUnit key.TimeUnit, incr int64) error
}

type CostEstimator interface {
	EstimatePromptCost(model string, tks int) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
}

func NewRecorder(s Store, c Cache, ce CostEstimator) *Recorder {
	return &Recorder{
		s:  s,
		c:  c,
		ce: ce,
	}
}

func (r *Recorder) RecordKeySpend(keyId string, model string, promptTks int, completionTks int, costLimitUnit key.TimeUnit) error {
	promptCost, err := r.ce.EstimatePromptCost(model, promptTks)
	if err != nil {
		return err
	}

	completionCost, err := r.ce.EstimateCompletionCost(model, completionTks)
	if err != nil {
		return err
	}

	micros := (promptCost + completionCost) * 1000000

	err = r.s.IncrementCounter(keyId, int64(micros))
	if err != nil {
		return err
	}

	if len(costLimitUnit) != 0 {
		err = r.c.IncrementCounter(keyId, costLimitUnit, int64(micros))
		if err != nil {
			return err
		}
	}

	return nil
}
