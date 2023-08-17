package recorder

type Recorder struct {
	s                     Store
	ce                    CostEstimator
	openAiTotalCostPrefix string
}

type Store interface {
	IncrementCounter(prefix string, keyId string, incr int64) error
}

type CostEstimator interface {
	EstimatePromptCost(model string, tks int) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
}

func NewRecorder(s Store, ce CostEstimator, openAiTotalCostPrefix string) *Recorder {
	return &Recorder{
		s:                     s,
		ce:                    ce,
		openAiTotalCostPrefix: openAiTotalCostPrefix,
	}
}

func (r *Recorder) RecordKeySpend(keyId string, model string, promptTks int, completionTks int) error {
	promptCost, err := r.ce.EstimatePromptCost(model, promptTks)
	if err != nil {
		return err
	}

	completionCost, err := r.ce.EstimateCompletionCost(model, completionTks)
	if err != nil {
		return err
	}

	micros := (promptCost + completionCost) * 1000000
	err = r.s.IncrementCounter(r.openAiTotalCostPrefix, keyId, int64(micros))
	if err != nil {
		return err
	}

	return nil
}
