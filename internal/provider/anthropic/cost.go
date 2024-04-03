package anthropic

import (
	"errors"
	"fmt"
	"strings"
)

var AnthropicPerMillionTokenCost = map[string]map[string]float64{
	"prompt": {
		"claude-instant":  0.8,
		"claude":          8,
		"claude-3-opus":   15,
		"claude-3-sonnet": 3,
		"claude-3-haiku":  0.25,
	},
	"completion": {
		"claude-instant":  2.4,
		"claude":          24,
		"claude-3-opus":   75,
		"claude-3-sonnet": 15,
		"claude-3-haiku":  1.25,
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
	if strings.HasPrefix(model, "claude-3-opus") {
		return "claude-3-opus"
	} else if strings.HasPrefix(model, "claude-3-sonnet") {
		return "claude-3-sonnet"
	} else if strings.HasPrefix(model, "claude-3-haiku") {
		return "claude-3-haiku"
	} else if strings.HasPrefix(model, "claude-instant") {
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

var (
	anthropicMessageOverhead = 4
)

func (ce *CostEstimator) CountMessagesTokens(messages []Message) int {
	count := 0

	for _, message := range messages {
		count += ce.tc.Count(message.Content) + anthropicMessageOverhead
	}

	return count + anthropicMessageOverhead
}
