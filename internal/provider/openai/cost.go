package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	goopenai "github.com/sashabaranov/go-openai"
)

var OpenAiPerThousandTokenCost = map[string]map[string]float64{
	"prompt": {
		"gpt-4-1106-preview":        0.01,
		"gpt-4-1106-vision-preview": 0.01,
		"gpt-4":                     0.03,
		"gpt-4-0314":                0.03,
		"gpt-4-0613":                0.03,
		"gpt-4-32k":                 0.06,
		"gpt-4-32k-0613":            0.06,
		"gpt-4-32k-0314":            0.06,
		"gpt-3.5-turbo":             0.0015,
		"gpt-3.5-turbo-1106":        0.001,
		"gpt-3.5-turbo-0301":        0.0015,
		"gpt-3.5-turbo-instruct":    0.0015,
		"gpt-3.5-turbo-0613":        0.0015,
		"gpt-3.5-turbo-16k":         0.0015,
		"gpt-3.5-turbo-16k-0613":    0.0015,
		"text-davinci-003":          0.12,
		"text-davinci-002":          0.12,
		"code-davinci-002":          0.12,
		"text-curie-001":            0.012,
		"text-babbage-001":          0.0024,
		"text-ada-001":              0.0016,
		"davinci":                   0.12,
		"curie":                     0.012,
		"babbage":                   0.0024,
		"ada":                       0.0016,
	},
	"fine_tune": {
		"text-davinci-003": 0.03,
		"text-davinci-002": 0.03,
		"code-davinci-002": 0.03,
		"text-curie-001":   0.03,
		"text-babbage-001": 0.0006,
		"text-ada-001":     0.0004,
		"davinci":          0.03,
		"curie":            0.03,
		"babbage":          0.0006,
		"ada":              0.0004,
	},
	"embeddings": {
		"text-embedding-ada-002":        0.0001,
		"text-similarity-ada-001":       0.004,
		"text-search-ada-doc-001":       0.004,
		"text-search-ada-query-001":     0.004,
		"code-search-ada-code-001":      0.004,
		"code-search-ada-text-001":      0.004,
		"code-search-babbage-code-001":  0.005,
		"code-search-babbage-text-001":  0.005,
		"text-similarity-babbage-001":   0.005,
		"text-search-babbage-doc-001":   0.005,
		"text-search-babbage-query-001": 0.005,
		"text-similarity-curie-001":     0.02,
		"text-search-curie-doc-001":     0.02,
		"text-search-curie-query-001":   0.02,
		"text-search-davinci-doc-001":   0.2,
		"text-search-davinci-query-001": 0.2,
		"text-similarity-davinci-001":   0.2,
	},
	"completion": {
		"gpt-3.5-turbo-1106":        0.002,
		"gpt-4-1106-preview":        0.03,
		"gpt-4-1106-vision-preview": 0.03,
		"gpt-4":                     0.06,
		"gpt-4-0314":                0.06,
		"gpt-4-0613":                0.06,
		"gpt-4-32k":                 0.12,
		"gpt-4-32k-0613":            0.12,
		"gpt-4-32k-0314":            0.12,
		"gpt-3.5-turbo":             0.002,
		"gpt-3.5-turbo-0301":        0.002,
		"gpt-3.5-turbo-0613":        0.002,
		"gpt-3.5-turbo-instruct":    0.002,
		"gpt-3.5-turbo-16k":         0.004,
		"gpt-3.5-turbo-16k-0613":    0.004,
	},
}

type tokenCounter interface {
	Count(model string, input string) (int, error)
}

type CostEstimator struct {
	tokenCostMap map[string]map[string]float64
	tc           tokenCounter
}

func NewCostEstimator(m map[string]map[string]float64, tc tokenCounter) *CostEstimator {
	return &CostEstimator{
		tokenCostMap: m,
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

func (ce *CostEstimator) EstimateChatCompletionPromptCost(r *goopenai.ChatCompletionRequest) (float64, error) {
	if len(r.Model) == 0 {
		return 0, errors.New("model is not provided")
	}

	tks, err := countTotalTokens(r, ce.tc)
	if err != nil {
		return 0, err
	}
	return ce.EstimatePromptCost(r.Model, tks)
}

func (ce *CostEstimator) EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error) {
	if len(r.Model.String()) == 0 {
		return 0, errors.New("model is not provided")
	}

	if inputs, ok := r.Input.([]string); ok {
		total := 0
		for _, input := range inputs {
			tks, err := ce.tc.Count(r.Model.String(), input)
			if err != nil {
				return 0, err
			}

			total += tks
		}

		return ce.EstimateEmbeddingsInputCost(r.Model.String(), total)
	} else if input, ok := r.Input.(string); ok {
		tks, err := ce.tc.Count(r.Model.String(), input)
		if err != nil {
			return 0, err
		}

		return ce.EstimateEmbeddingsInputCost(r.Model.String(), tks)
	}

	return 0, errors.New("input format is not recognized")
}

func countFunctionTokens(r *goopenai.ChatCompletionRequest, tc tokenCounter) (int, error) {
	if len(r.Functions) == 0 {
		return 0, nil
	}

	defs := formatFunctionDefinitions(r)
	tks, err := tc.Count(r.Model, defs)
	if err != nil {
		return 0, err
	}

	tks += 9
	return tks, nil
}

func formatFunctionDefinitions(r *goopenai.ChatCompletionRequest) string {
	lines := []string{
		"namespace functions {", "",
	}

	for _, f := range r.Functions {
		if len(f.Description) != 0 {
			lines = append(lines, fmt.Sprintf("// %s", f.Description))
		}

		if f.Parameters != nil {
			lines = append(lines, fmt.Sprintf("type %s = (_: {`", f.Name))

			params := &FuntionCallProp{}
			data, err := json.Marshal(f.Parameters)
			if err == nil {
				err := json.Unmarshal(data, params)
				if err == nil {
					lines = append(lines, formatObjectProperties(params, 0))
				}
			}

			lines = append(lines, "}) => any;")
		}

		if f.Parameters == nil {
			lines = append(lines, fmt.Sprintf("type %s = () => any;", f.Name))
		}

		lines = append(lines, "")
	}

	lines = append(lines, "} // namespace functions")
	return strings.Join(lines, "\n")
}

func countMessageTokens(r *goopenai.ChatCompletionRequest, tc tokenCounter) (int, error) {
	if len(r.Messages) == 0 {
		return 0, nil
	}

	result := 0
	padded := false

	for _, msg := range r.Messages {
		content := msg.Content
		if msg.Role == "system" && !padded {
			content += "\n"
			padded = true
		}

		contentTks, err := tc.Count(r.Model, content)
		if err != nil {
			return 0, err
		}

		roleTks, err := tc.Count(r.Model, msg.Role)
		if err != nil {
			return 0, err
		}

		nameTks, err := tc.Count(r.Model, msg.Name)
		if err != nil {
			return 0, err
		}

		result += contentTks
		result += roleTks
		result += nameTks

		result += 3
		if len(msg.Name) != 0 {
			result += 1
		}

		if msg.Role == "function" {
			result -= 2
		}

		if msg.FunctionCall != nil {
			result += 3
		}
	}

	return result, nil
}

func countTotalTokens(r *goopenai.ChatCompletionRequest, tc tokenCounter) (int, error) {
	if r == nil {
		return 0, nil
	}

	tks := 3

	ftks, err := countFunctionTokens(r, tc)
	if err != nil {
		return 0, err
	}

	mtks, err := countMessageTokens(r, tc)
	if err != nil {
		return 0, err
	}

	systemExists := false
	for _, msg := range r.Messages {
		if msg.Role == "system" {
			systemExists = true
		}

	}

	if len(r.Functions) != 0 && systemExists {
		tks -= 4
	}

	return tks + ftks + mtks, err
}
