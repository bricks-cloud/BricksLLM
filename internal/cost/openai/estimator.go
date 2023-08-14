package openai

import (
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

type CostEstimator struct{}

func NewCostEstiamtor() (*CostEstimator, error) {
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())

	return &CostEstimator{}, nil
}

func (ce *CostEstimator) estimateTokens(model string, input string) (int, error) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return 0, err
	}

	token := tkm.Encode(input, nil, nil)
	return len(token), nil
}

func (ce *CostEstimator) EstimatePromptTokenCost(model string, input string) (float64, error) {
	num, err := ce.estimateTokens(model, input)
	if err != nil {
		return 0, err
	}

	cost, err := estimatePromptTokenCost(model, num)
	if err != nil {
		return 0, err
	}

	return cost, nil
}

func (ce *CostEstimator) EstimateTotalCost(model string, input string) (float64, error) {
	num, err := ce.estimateTokens(model, input)
	if err != nil {
		return 0, err
	}

	cost, err := estimatePromptTokenCost(model, num)
	if err != nil {
		return 0, err
	}

	return cost, nil
}

func estimateTotalCost(model string, promptTokens int, completionTokens int) (float64, error) {
	promptTokenCost, err := estimatePromptTokenCost(model, promptTokens)
	if err != nil {
		return 0, err
	}

	completionTokenCost, err := estimateCompletionTokenCost(model, completionTokens)
	if err != nil {
		return 0, err
	}

	return promptTokenCost + completionTokenCost, nil
}

func estimatePromptTokenCost(model string, promptTokens int) (float64, error) {
	if strings.HasPrefix(model, "gpt-4") {
		if strings.HasPrefix(model, "gpt-4-32k") {
			return float64(promptTokens) * 0.06, nil
		}

		return float64(promptTokens) * 0.03, nil
	}

	if strings.HasPrefix(model, "gpt-3.5-turbo") {
		if strings.HasPrefix(model, "gpt-3.5-turbo-16k") {
			return float64(promptTokens) * 0.003, nil
		}

		return float64(promptTokens) * 0.0015, nil
	}

	return 0, fmt.Errorf("openai model is not recognized: %s", model)
}

func estimateCompletionTokenCost(model string, completionTokens int) (float64, error) {
	if strings.HasPrefix(model, "gpt-4") {
		if strings.HasPrefix(model, "gpt-4-32k") {
			return float64(completionTokens) * 0.12, nil
		}

		return float64(completionTokens) * 0.06, nil
	}

	if strings.HasPrefix(model, "gpt-3.5-turbo") {
		if strings.HasPrefix(model, "gpt-3.5-turbo-16k") {
			return float64(completionTokens) * 0.004, nil
		}

		return float64(completionTokens) * 0.002, nil
	}

	return 0, fmt.Errorf("openai model is not recognized: %s", model)
}
