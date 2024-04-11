package deepinfra

import (
	"errors"
	"fmt"
	"strings"
)

var DeepinfraPerMillionTokenCost = map[string]map[string]float64{
	"prompt": {
		"baai/bge-large-en-v1.5":                              0.01,
		"baai/bge-base-en-v1.5":                               0.005,
		"intfloat/e5-base-v2":                                 0.005,
		"intfloat/e5-large-v2":                                0.01,
		"sentence-transformers/all-minilm-l12-v2":             0.005,
		"sentence-transformers/all-minilm-l6-v2":              0.005,
		"sentence-transformers/all-mpnet-base-v2":             0.005,
		"sentence-transformers/clip-vit-b-32":                 0.005,
		"sentence-transformers/clip-vit-b-32-multilingual-v1": 0.005,
		"sentence-transformers/multi-qa-mpnet-base-dot-v1":    0.005,
		"sentence-transformers/paraphrase-minilm-l6-v2":       0.005,
		"shibing624/text2vec-base-chinese":                    0.005,
		"thenlper/gte-base":                                   0.005,
		"thenlper/gte-large":                                  0.01,
	},
}

type CostEstimator struct {
	tokenCostMap map[string]map[string]float64
}

func NewCostEstimator() *CostEstimator {
	return &CostEstimator{
		tokenCostMap: DeepinfraPerMillionTokenCost,
	}
}

func (ce *CostEstimator) EstimateEmbeddingsInputCost(model string, tks int) (float64, error) {
	costMap, ok := ce.tokenCostMap["prompt"]
	if !ok {
		return 0, errors.New("prompt token cost is not provided")

	}

	lowerCased := strings.ToLower(model)
	cost, ok := costMap[lowerCased]
	if !ok {
		return 0, fmt.Errorf("%s is not present in the cost map provided", model)
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000000 * cost, nil
}
