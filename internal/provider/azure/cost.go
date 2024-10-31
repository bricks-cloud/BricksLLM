package azure

import (
	"errors"
	"fmt"

	"github.com/bricks-cloud/bricksllm/internal/util"
	goopenai "github.com/sashabaranov/go-openai"
)

var AzureOpenAiPerThousandTokenCost = map[string]map[string]float64{
	// updated according to this link:
	// https://azure.microsoft.com/en-gb/pricing/details/cognitive-services/openai-service/
	"prompt": {
		"gpt-4o":                 0.005,
		"gpt-4o-2024-08-16":      0.0025,
		"gpt-4o-2024-05-13":      0.005,
		"gpt-4-turbo":            0.01,
		"gpt-4-turbo-2024-04-09": 0.01,
		"gpt-4":                  0.03,
		"gpt-4-0125-preview":     0.03,
		"gpt-4-32k":              0.06,
		"gpt-4-vision":           0.06,
		"gpt-35-turbo":           0.0015,
		"gpt-35-turbo-instruct":  0.0015,
		"gpt-35-turbo-16k":       0.003,
	},
	"embeddings": {
		"ada":                    0.0001,
		"text-embedding-3-large": 0.00013,
		"text-embedding-3-small": 0.00002,
	},
	"completion": {
		"gpt-4o":                 0.015,
		"gpt-4o-2024-08-16":      0.01,
		"gpt-4o-2024-05-13":      0.015,
		"gpt-4-turbo":            0.03,
		"gpt-4-turbo-2024-04-09": 0.03,
		"gpt-4":                  0.06,
		"gpt-4-0125-preview":     0.06,
		"gpt-4-32k":              0.12,
		"gpt-4-vision":           0.12,
		"gpt-35-turbo":           0.002,
		"gpt-35-turbo-instruct":  0.002,
		"gpt-35-turbo-16k":       0.004,
	},
}

type CostEstimator struct {
	tokenCostMap map[string]map[string]float64
}

func NewCostEstimator() *CostEstimator {
	return &CostEstimator{
		tokenCostMap: AzureOpenAiPerThousandTokenCost,
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

	cost, ok := costMap[model]
	if !ok {
		return 0, fmt.Errorf("%s is not present in the cost map provided", model)
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000 * cost, nil
}

func (ce *CostEstimator) EstimateEmbeddingsInputCost(model string, tks int) (float64, error) {
	costMap, ok := ce.tokenCostMap["embeddings"]
	if !ok {
		return 0, errors.New("embeddings token cost is not provided")

	}

	cost, ok := costMap[model]
	if !ok {
		return 0, fmt.Errorf("%s is not present in the cost map provided", model)
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000 * cost, nil
}

func (ce *CostEstimator) EstimateCompletionCost(model string, tks int) (float64, error) {
	costMap, ok := ce.tokenCostMap["completion"]
	if !ok {
		return 0, errors.New("prompt token cost is not provided")
	}

	cost, ok := costMap[model]
	if !ok {
		return 0, errors.New("model is not present in the cost map provided")
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000 * cost, nil
}

func (ce *CostEstimator) EstimateChatCompletionStreamCostWithTokenCounts(model string, content string) (int, float64, error) {
	if len(model) == 0 {
		return 0, 0, errors.New("model is not provided")
	}

	tks, err := Count(model, content)
	if err != nil {
		return 0, 0, err
	}

	cost, err := ce.EstimateCompletionCost(model, tks)
	if err != nil {
		return 0, 0, err
	}

	return tks, cost, nil
}

func (ce *CostEstimator) EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error) {
	model := "ada"

	input, err := util.ConvertAnyToStr(r.Input)
	if err != nil {
		return 0, err
	}

	tks, err := Count(model, input)
	if err != nil {
		return 0, err
	}

	return ce.EstimateEmbeddingsInputCost(model, tks)
}
