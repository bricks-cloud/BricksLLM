package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/util"
	goopenai "github.com/sashabaranov/go-openai"
)

func useFinetuneModel(model string) string {
	if isFinetuneModel(model) {
		return parseFinetuneModel(model)
	}

	return model
}

func isFinetuneModel(model string) bool {
	return strings.HasPrefix(model, "ft:")
}

func parseFinetuneModel(model string) string {
	parts := strings.Split(model, ":")
	if len(parts) > 2 {
		return "finetune-" + parts[1]
	}

	return model
}

var OpenAiPerThousandTokenCost = map[string]map[string]float64{
	"prompt": {
		"gpt-4o":                      0.005,
		"gpt-4o-mini":                 0.00015,
		"gpt-4o-mini-2024-07-18":      0.00015,
		"gpt-4o-2024-05-13":           0.005,
		"gpt-4-1106-preview":          0.01,
		"gpt-4-turbo-preview":         0.01,
		"gpt-4-turbo":                 0.01,
		"gpt-4-turbo-2024-04-09":      0.01,
		"gpt-4-0125-preview":          0.01,
		"gpt-4-1106-vision-preview":   0.01,
		"gpt-4-vision-preview":        0.01,
		"gpt-4":                       0.03,
		"gpt-4-0314":                  0.03,
		"gpt-4-0613":                  0.03,
		"gpt-4-32k":                   0.06,
		"gpt-4-32k-0613":              0.06,
		"gpt-4-32k-0314":              0.06,
		"gpt-3.5-turbo":               0.0015,
		"gpt-3.5-turbo-1106":          0.001,
		"gpt-3.5-turbo-0125":          0.0005,
		"gpt-3.5-turbo-0301":          0.0015,
		"gpt-3.5-turbo-instruct":      0.0015,
		"gpt-3.5-turbo-0613":          0.0015,
		"gpt-3.5-turbo-16k":           0.0015,
		"gpt-3.5-turbo-16k-0613":      0.0015,
		"text-davinci-003":            0.12,
		"text-davinci-002":            0.12,
		"code-davinci-002":            0.12,
		"text-curie-001":              0.012,
		"text-babbage-001":            0.0024,
		"text-ada-001":                0.0016,
		"davinci":                     0.12,
		"curie":                       0.012,
		"babbage":                     0.0024,
		"ada":                         0.0016,
		"finetune-gpt-4-0613":         0.045,
		"finetune-gpt-3.5-turbo-0125": 0.003,
		"finetune-gpt-3.5-turbo-1106": 0.003,
		"finetune-gpt-3.5-turbo-0613": 0.003,
		"finetune-babbage-002":        0.0016,
		"finetune-davinci-002":        0.012,
	},
	"finetune": {
		"gpt-4-0613":         0.09,
		"gpt-3.5-turbo-0125": 0.008,
		"gpt-3.5-turbo-1106": 0.008,
		"gpt-3.5-turbo-0613": 0.008,
		"babbage-002":        0.0004,
		"davinci-002":        0.006,
	},
	"embeddings": {
		"text-embedding-ada-002": 0.0001,
		"text-embedding-3-small": 0.00002,
		"text-embedding-3-large": 0.00013,
	},
	"audio": {
		"whisper-1": 0.006,
		"tts-1":     0.015,
		"tts-1-hd":  0.03,
	},
	"completion": {
		"gpt-3.5-turbo-1106":          0.002,
		"gpt-4o":                      0.015,
		"gpt-4o-mini":                 0.0006,
		"gpt-4o-mini-2024-07-18":      0.0006,
		"gpt-4o-2024-05-13":           0.015,
		"gpt-4-turbo-preview":         0.03,
		"gpt-4-turbo":                 0.03,
		"gpt-4-turbo-2024-04-09":      0.03,
		"gpt-4-1106-preview":          0.03,
		"gpt-4-0125-preview":          0.03,
		"gpt-4-1106-vision-preview":   0.03,
		"gpt-4-vision-preview":        0.03,
		"gpt-4":                       0.06,
		"gpt-4-0314":                  0.06,
		"gpt-4-0613":                  0.06,
		"gpt-4-32k":                   0.12,
		"gpt-4-32k-0613":              0.12,
		"gpt-4-32k-0314":              0.12,
		"gpt-3.5-turbo":               0.002,
		"gpt-3.5-turbo-0125":          0.0015,
		"gpt-3.5-turbo-0301":          0.002,
		"gpt-3.5-turbo-0613":          0.002,
		"gpt-3.5-turbo-instruct":      0.002,
		"gpt-3.5-turbo-16k":           0.004,
		"gpt-3.5-turbo-16k-0613":      0.004,
		"finetune-gpt-4-0613":         0.09,
		"finetune-gpt-3.5-turbo-0125": 0.006,
		"finetune-gpt-3.5-turbo-1106": 0.006,
		"finetune-gpt-3.5-turbo-0613": 0.006,
		"finetune-babbage-002":        0.0016,
		"finetune-davinci-002":        0.012,
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

	cost, ok := costMap[useFinetuneModel(model)]
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

	cost, ok := costMap[useFinetuneModel(model)]
	if !ok {
		return 0, errors.New("model is not present in the cost map provided")
	}

	tksInFloat := float64(tks)
	return tksInFloat / 1000 * cost, nil
}

func (ce *CostEstimator) EstimateChatCompletionPromptTokenCounts(model string, r *goopenai.ChatCompletionRequest) (int, error) {
	tks, err := countTotalTokens(model, r, ce.tc)
	if err != nil {
		return 0, err
	}

	return tks, nil
}

func (ce *CostEstimator) EstimateChatCompletionPromptCostWithTokenCounts(r *goopenai.ChatCompletionRequest) (int, float64, error) {
	if len(r.Model) == 0 {
		return 0, 0, errors.New("model is not provided")
	}

	tks, err := countTotalTokens(r.Model, r, ce.tc)
	if err != nil {
		return 0, 0, err
	}

	cost, err := ce.EstimatePromptCost(r.Model, tks)
	if err != nil {
		return 0, 0, err
	}

	return tks, cost, nil
}

func (ce *CostEstimator) EstimateChatCompletionStreamCostWithTokenCounts(model string, content string) (int, float64, error) {
	if len(model) == 0 {
		return 0, 0, errors.New("model is not provided")
	}

	tks, err := ce.tc.Count(model, content)
	if err != nil {
		return 0, 0, err
	}

	cost, err := ce.EstimateCompletionCost(model, tks)
	if err != nil {
		return 0, 0, err
	}

	return tks, cost, nil
}

func (ce *CostEstimator) EstimateCompletionsRequestCostWithTokenCounts(model string, content any) (int, float64, error) {
	if len(model) == 0 {
		return 0, 0, errors.New("model is not provided")
	}

	input, err := util.ConvertAnyToStr(content)
	if err != nil {
		return 0, 0, err
	}

	tks, err := ce.tc.Count(model, input)
	if err != nil {
		return 0, 0, err
	}

	cost, err := ce.EstimatePromptCost(model, tks)
	if err != nil {
		return 0, 0, err
	}

	return tks, cost, nil
}

func (ce *CostEstimator) EstimateCompletionsStreamCostWithTokenCounts(model string, content string) (int, float64, error) {
	if len(model) == 0 {
		return 0, 0, errors.New("model is not provided")
	}

	tks, err := ce.tc.Count(model, content)
	if err != nil {
		return 0, 0, err
	}

	cost, err := ce.EstimateCompletionCost(model, tks)
	if err != nil {
		return 0, 0, err
	}

	return tks, cost, nil
}

func (ce *CostEstimator) EstimateTranscriptionCost(secs float64, model string) (float64, error) {
	costMap, ok := ce.tokenCostMap["audio"]
	if !ok {
		return 0, errors.New("audio cost map is not provided")
	}

	cost, ok := costMap[model]
	if !ok {
		return 0, errors.New("model is not present in the audio cost map")
	}

	return math.Trunc(secs) / 60 * cost, nil
}

func (ce *CostEstimator) EstimateSpeechCost(input string, model string) (float64, error) {
	costMap, ok := ce.tokenCostMap["audio"]
	if !ok {
		return 0, errors.New("audio cost map is not provided")
	}

	cost, ok := costMap[model]
	if !ok {
		return 0, errors.New("model is not present in the audio cost map")
	}

	return float64(len(input)) / 1000 * cost, nil
}

func (ce *CostEstimator) EstimateFinetuningCost(num int, model string) (float64, error) {
	costMap, ok := ce.tokenCostMap["finetune"]
	if !ok {
		return 0, errors.New("audio cost map is not provided")
	}

	cost, ok := costMap[model]
	if !ok {
		return 0, errors.New("model is not present in the audio cost map")
	}

	return cost * float64(num), nil
}

func (ce *CostEstimator) EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error) {
	if len(string(r.Model)) == 0 {
		return 0, errors.New("model is not provided")
	}

	input, err := util.ConvertAnyToStr(r.Input)
	if err != nil {
		return 0, err
	}

	tks, err := ce.tc.Count(string(r.Model), input)
	if err != nil {
		return 0, err
	}

	return ce.EstimateEmbeddingsInputCost(string(r.Model), tks)
}

func countFunctionTokens(model string, r *goopenai.ChatCompletionRequest, tc tokenCounter) (int, error) {
	if len(r.Functions) == 0 {
		return 0, nil
	}

	defs := formatFunctionDefinitions(r)
	tks, err := tc.Count(model, defs)
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

func countMessageTokens(model string, r *goopenai.ChatCompletionRequest, tc tokenCounter) (int, error) {
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

		contentTks, err := tc.Count(model, content)
		if err != nil {
			return 0, err
		}

		roleTks, err := tc.Count(model, msg.Role)
		if err != nil {
			return 0, err
		}

		nameTks, err := tc.Count(model, msg.Name)
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

func countTotalTokens(model string, r *goopenai.ChatCompletionRequest, tc tokenCounter) (int, error) {
	if r == nil {
		return 0, nil
	}

	tks := 3

	ftks, err := countFunctionTokens(model, r, tc)
	if err != nil {
		return 0, err
	}

	mtks, err := countMessageTokens(model, r, tc)
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
