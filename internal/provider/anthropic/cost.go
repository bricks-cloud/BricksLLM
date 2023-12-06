package anthropic

import (
	"errors"
	"fmt"
	"strings"
)

var AnthropicPerMillionTokenCost = map[string]map[string]float64{
	"prompt": {
		"claude-instant": 0.8,
		"claude":         8,
	},
	"completion": {
		"claude-instant": 2.4,
		"claude":         24,
	},
}

type tokenCounter interface {
	Count(input string) int
}

type CostEstimator struct {
	tokenCostMap map[string]map[string]float64
	tc           tokenCounter
}

func NewCostEstimator(tc tokenCounter) *CostEstimator {
	return &CostEstimator{
		tokenCostMap: AnthropicPerMillionTokenCost,
		tc:           tc,
	}
}

func (ce *CostEstimator) EstimateTotalCost(model string, promptTks, completionTks int) (float64, error) {
	promptCost, err := ce.EstimatePromptCost(model, promptTks)
	if err != nil {
		return 0, err
	}

	completionCost, err := ce.EstimateCompletionCost(model, completionTks)
	if err != nil {
		return 0, err
	}

	return promptCost + completionCost, nil
}

func (ce *CostEstimator) EstimatePromptCost(model string, tks int) (float64, error) {
	costMap, ok := ce.tokenCostMap["prompt"]
	if !ok {
		return 0, errors.New("prompt token cost is not provided")

	}

	selected := selectModel(model)
	cost, ok := costMap[selected]
	if !ok {
		return 0, fmt.Errorf("%s is not present in the cost map provided", model)
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000000 * cost, nil
}

func selectModel(model string) string {
	if strings.HasPrefix(model, "claude-instant") {
		return "claude-instant"
	} else if strings.HasPrefix(model, "claude") {
		return "claude"
	}

	return ""
}

func (ce *CostEstimator) EstimateCompletionCost(model string, tks int) (float64, error) {
	costMap, ok := ce.tokenCostMap["completion"]
	if !ok {
		return 0, errors.New("prompt token cost is not provided")
	}

	selected := selectModel(model)
	cost, ok := costMap[selected]
	if !ok {
		return 0, errors.New("model is not present in the cost map provided")
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000000 * cost, nil
}

func (ce *CostEstimator) Count(input string) int {
	return ce.tc.Count(input)
}
