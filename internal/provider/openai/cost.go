package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/util"
	responsesOpenai "github.com/openai/openai-go/v3/responses"
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
		"gpt-image-1.5":        0.005,
		"gpt-image-1":          0.005,
		"chatgpt-image-latest": 0.005,
		"gpt-image-1-mini":     0.002,

		"gpt-6-astra":       0.020,
		"gpt-6-astra~long":  0.020,
		"gpt-6-astra~short": 0.010,
		"gpt-6.1-sol":       0.004,
		"gpt-6.1-sol~long":  0.004,
		"gpt-6.1-sol~short": 0.002,
		"gpt-6-luna":        0.0002,
		"gpt-6-luna~long":   0.0002,
		"gpt-6-luna~short":  0.0001,

		"gpt-5.6-sol":       0.008,
		"gpt-5.6-sol~long":  0.008,
		"gpt-5.6-sol~short": 0.004,
		"gpt-5.6-cyber":     0.0125,

		"gpt-5.5":           0.01,
		"gpt-5.5~long":      0.01,
		"gpt-5.5~short":     0.005,
		"gpt-5.5-pro":       0.06,
		"gpt-5.5-pro~long":  0.06,
		"gpt-5.5-pro~short": 0.03,

		"gpt-5.4":           0.005,
		"gpt-5.4~long":      0.005,
		"gpt-5.4~short":     0.0025,
		"gpt-5.4-mini":      0.00075,
		"gpt-5.4-nano":      0.0002,
		"gpt-5.4-pro":       0.06,
		"gpt-5.4-pro~long":  0.06,
		"gpt-5.4-pro~short": 0.03,

		"gpt-5.3-codex": 0.00175,

		"gpt-5.2-codex": 0.00175,

		"gpt-5.1-codex-mini": 0.00025,

		"gpt-5.2-chat-latest":          0.001750,
		"gpt-5.1-chat-latest":          0.001250,
		"gpt-5.1-codex-max":            0.001250,
		"gpt-5.1-codex":                0.001250,
		"gpt-5.2-pro":                  0.021000,
		"gpt-5.2":                      0.001750,
		"gpt-5.1":                      0.001250,
		"gpt-5":                        0.001250,
		"gpt-5-mini":                   0.000250,
		"gpt-5-nano":                   0.000050,
		"gpt-5-chat-latest":            0.001250,
		"gpt-5-codex":                  0.001250,
		"gpt-5-pro":                    0.015000,
		"gpt-4.1":                      0.002000,
		"gpt-4.1-mini":                 0.000400,
		"gpt-4.1-nano":                 0.000100,
		"gpt-4o":                       0.002500,
		"gpt-4o-2024-05-13":            0.005000,
		"gpt-4o-mini":                  0.000150,
		"gpt-realtime":                 0.004000,
		"gpt-realtime-mini":            0.000600,
		"gpt-4o-realtime-preview":      0.005000,
		"gpt-4o-mini-realtime-preview": 0.000600,
		"gpt-audio":                    0.002500,
		"gpt-audio-mini":               0.000600,
		"gpt-4o-audio-preview":         0.002500,
		"gpt-4o-mini-audio-preview":    0.000150,
		"o1":                           0.015000,
		"o1-pro":                       0.150000,
		"o3-pro":                       0.020000,
		"o3":                           0.002000,
		"o3-deep-research":             0.010000,
		"o4-mini":                      0.001100,
		"o4-mini-deep-research":        0.002000,
		"o3-mini":                      0.001100,
		"o1-mini":                      0.001100,
		"codex-mini-latest":            0.001500,
		"gpt-4o-mini-search-preview":   0.000150,
		"gpt-4o-search-preview":        0.002500,
		"computer-use-preview":         0.003000,
		"chatgpt-4o-latest":            0.005000,
		"gpt-4-turbo-2024-04-09":       0.010000,
		"gpt-4-0125-preview":           0.010000,
		"gpt-4-1106-preview":           0.010000,
		"gpt-4-1106-vision-preview":    0.010000,
		"gpt-4-0613":                   0.030000,
		"gpt-4-0314":                   0.030000,
		"gpt-4-32k":                    0.060000,
		"gpt-3.5-turbo":                0.000500,
		"gpt-3.5-turbo-0125":           0.000500,
		"gpt-3.5-turbo-1106":           0.001000,
		"gpt-3.5-turbo-0613":           0.001500,
		"gpt-3.5-0301":                 0.001500,
		"gpt-3.5-turbo-instruct":       0.001500,
		"gpt-3.5-turbo-16k-0613":       0.003000,
		"davinci-002":                  0.002000,
		"babbage-002":                  0.000400,
	},
	"cached-prompt": {
		"gpt-6-astra":       0.002,
		"gpt-6-astra~long":  0.002,
		"gpt-6-astra~short": 0.001,
		"gpt-6.1-sol":       0.0002,
		"gpt-6.1-sol~long":  0.0002,
		"gpt-6.1-sol~short": 0.0001,
		"gpt-6-luna":        0.00002,
		"gpt-6-luna~long":   0.00002,
		"gpt-6-luna~short":  0.00001,

		"gpt-5.6-sol":       0.0008,
		"gpt-5.6-sol~long":  0.0008,
		"gpt-5.6-sol~short": 0.0004,
		"gpt-5.6-cyber":     0.00125,

		"gpt-5.5":       0.001,
		"gpt-5.5~long":  0.001,
		"gpt-5.5~short": 0.0005,

		"gpt-5.4":       0.0005,
		"gpt-5.4~long":  0.0005,
		"gpt-5.4~short": 0.00025,
		"gpt-5.4-mini":  0.000075,
		"gpt-5.4-nano":  0.00002,

		"gpt-image-1.5":        0.00125,
		"gpt-image-1":          0.00125,
		"chatgpt-image-latest": 0.00125,
		"gpt-image-1-mini":     0.0002,

		"gpt-5.2-chat-latest":          0.000175,
		"gpt-5.1-chat-latest":          0.000125,
		"gpt-5.1-codex-max":            0.000125,
		"gpt-5.1-codex":                0.000125,
		"gpt-5.2":                      0.000175,
		"gpt-5.1":                      0.000125,
		"gpt-5":                        0.000125,
		"gpt-5-mini":                   0.000025,
		"gpt-5-nano":                   0.000005,
		"gpt-5-chat-latest":            0.000125,
		"gpt-5-codex":                  0.000125,
		"gpt-4.1":                      0.000500,
		"gpt-4.1-mini":                 0.000100,
		"gpt-4.1-nano":                 0.000025,
		"gpt-4o":                       0.001250,
		"gpt-4o-mini":                  0.000075,
		"gpt-realtime":                 0.000400,
		"gpt-realtime-mini":            0.000060,
		"gpt-4o-realtime-preview":      0.002500,
		"gpt-4o-mini-realtime-preview": 0.000300,
		"o1":                           0.007500,
		"o3":                           0.000500,
		"o3-deep-research":             0.002500,
		"o4-mini":                      0.000275,
		"o4-mini-deep-research":        0.000500,
		"o3-mini":                      0.000550,
		"o1-mini":                      0.000550,
		"codex-mini-latest":            0.000375,
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

		"tts-1":    0.015,
		"tts-1-hd": 0.03,

		"gpt-4o-transcribe":         0.006,
		"gpt-4o-transcribe-diarize": 0.006,
		"gpt-4o-mini-transcribe":    0.003,

		"gpt-4o-mini-tts": 0.012,
	},
	"transcription-input": {
		"gpt-4o-transcribe":         0.0025,
		"gpt-4o-transcribe-diarize": 0.0025,
		"gpt-4o-mini-transcribe":    0.00125,
	},
	"transcription-output": {
		"gpt-4o-transcribe":         0.01,
		"gpt-4o-transcribe-diarize": 0.01,
		"gpt-4o-mini-transcribe":    0.005,
	},
	"video": { // $ per sec
		"sora-2":          0.1,
		"sora-2-pro":      0.30,
		"sora-2-720":      0.1,
		"sora-2-pro-720":  0.30,
		"sora-2-pro-1024": 0.5,
		"sora-2-pro-1080": 0.7,
	},
	"completion": {
		"gpt-image-1.5":        0.010,
		"chatgpt-image-latest": 0.010,

		"gpt-6-astra":       0.075,
		"gpt-6-astra~long":  0.075,
		"gpt-6-astra~short": 0.05,
		"gpt-6.1-sol":       0.015,
		"gpt-6.1-sol~long":  0.015,
		"gpt-6.1-sol~short": 0.01,
		"gpt-6-luna":        0.00075,
		"gpt-6-luna~long":   0.00075,
		"gpt-6-luna~short":  0.0005,

		"gpt-5.6-sol":       0.03,
		"gpt-5.6-sol~long":  0.03,
		"gpt-5.6-sol~short": 0.02,
		"gpt-5.6-cyber":     0.075,

		"gpt-5.5":           0.045,
		"gpt-5.5~long":      0.045,
		"gpt-5.5~short":     0.03,
		"gpt-5.5-pro":       0.27,
		"gpt-5.5-pro~long":  0.27,
		"gpt-5.5-pro~short": 0.18,

		"gpt-5.4":           0.0225,
		"gpt-5.4~long":      0.0225,
		"gpt-5.4~short":     0.015,
		"gpt-5.4-mini":      0.0045,
		"gpt-5.4-nano":      0.00125,
		"gpt-5.4-pro":       0.27,
		"gpt-5.4-pro~long":  0.27,
		"gpt-5.4-pro~short": 0.18,

		"gpt-5.3-codex": 0.014,

		"gpt-5.2-codex": 0.014,

		"gpt-5.1-codex-mini": 0.002,

		"gpt-5.2-chat-latest":          0.014000,
		"gpt-5.1-chat-latest":          0.010000,
		"gpt-5.1-codex-max":            0.010000,
		"gpt-5.1-codex":                0.010000,
		"gpt-5.2-pro":                  0.168000,
		"gpt-5.2":                      0.014000,
		"gpt-5.1":                      0.010000,
		"gpt-5":                        0.010000,
		"gpt-5-mini":                   0.002000,
		"gpt-5-nano":                   0.000400,
		"gpt-5-chat-latest":            0.010000,
		"gpt-5-codex":                  0.010000,
		"gpt-5-pro":                    0.120000,
		"gpt-4.1":                      0.008000,
		"gpt-4.1-mini":                 0.001600,
		"gpt-4.1-nano":                 0.000400,
		"gpt-4o":                       0.010000,
		"gpt-4o-2024-05-13":            0.015000,
		"gpt-4o-mini":                  0.000600,
		"gpt-realtime":                 0.016000,
		"gpt-realtime-mini":            0.002400,
		"gpt-4o-realtime-preview":      0.020000,
		"gpt-4o-mini-realtime-preview": 0.002400,
		"gpt-audio":                    0.010000,
		"gpt-audio-mini":               0.002400,
		"gpt-4o-audio-preview":         0.010000,
		"gpt-4o-mini-audio-preview":    0.000600,
		"o1":                           0.060000,
		"o1-pro":                       0.600000,
		"o3-pro":                       0.080000,
		"o3":                           0.008000,
		"o3-deep-research":             0.040000,
		"o4-mini":                      0.004400,
		"o4-mini-deep-research":        0.008000,
		"o3-mini":                      0.004400,
		"o1-mini":                      0.004400,
		"codex-mini-latest":            0.006000,
		"gpt-4o-mini-search-preview":   0.000600,
		"gpt-4o-search-preview":        0.010000,
		"computer-use-preview":         0.012000,
		"chatgpt-4o-latest":            0.015000,
		"gpt-4-turbo-2024-04-09":       0.030000,
		"gpt-4-0125-preview":           0.030000,
		"gpt-4-1106-preview":           0.030000,
		"gpt-4-1106-vision-preview":    0.030000,
		"gpt-4-0613":                   0.060000,
		"gpt-4-0314":                   0.060000,
		"gpt-4-32k":                    0.120000,
		"gpt-3.5-turbo":                0.001500,
		"gpt-3.5-turbo-0125":           0.001500,
		"gpt-3.5-turbo-1106":           0.002000,
		"gpt-3.5-turbo-0613":           0.002000,
		"gpt-3.5-0301":                 0.002000,
		"gpt-3.5-turbo-instruct":       0.002000,
		"gpt-3.5-turbo-16k-0613":       0.004000,
		"davinci-002":                  0.002000,
		"babbage-002":                  0.000400,
	},
	"images": {
		"dall-e-2":      0.02,
		"dall-e-2-256":  0.016,
		"dall-e-2-512":  0.018,
		"dall-e-2-1024": 0.02,

		"dall-e-3":               0.04,
		"dall-e-3-1024-standart": 0.04,
		"dall-e-3-1792-standart": 0.08,
		"dall-e-3-1024-hd":       0.08,
		"dall-e-3-1792-hd":       0.12,

		"gpt-image-1.5-1536-high":   0.2,
		"gpt-image-1.5-1536-medium": 0.05,
		"gpt-image-1.5-1536-low":    0.013,
		"gpt-image-1.5-1024-high":   0.133,
		"gpt-image-1.5-1024-medium": 0.034,
		"gpt-image-1.5-1024-low":    0.009,

		"chatgpt-image-latest-1536-high":   0.2,
		"chatgpt-image-latest-1536-medium": 0.05,
		"chatgpt-image-latest-1536-low":    0.013,
		"chatgpt-image-latest-1024-high":   0.133,
		"chatgpt-image-latest-1024-medium": 0.034,
		"chatgpt-image-latest-1024-low":    0.009,

		"gpt-image-1-1536-high":   0.25,
		"gpt-image-1-1536-medium": 0.063,
		"gpt-image-1-1536-low":    0.016,
		"gpt-image-1-1024-high":   0.167,
		"gpt-image-1-1024-medium": 0.042,
		"gpt-image-1-1024-low":    0.011,

		"gpt-image-1-mini-1536-high":   0.052,
		"gpt-image-1-mini-1536-medium": 0.015,
		"gpt-image-1-mini-1536-low":    0.006,
		"gpt-image-1-mini-1024-high":   0.036,
		"gpt-image-1-mini-1024-medium": 0.011,
		"gpt-image-1-mini-1024-low":    0.005,
	},
	"images-tokens-input": {
		"gpt-image-1.5":        0.008,
		"gpt-image-1":          0.010,
		"chatgpt-image-latest": 0.008,
		"gpt-image-1-mini":     0.0025,
	},
	"images-tokens-cached-input": {
		"gpt-image-1.5":        0.002,
		"gpt-image-1":          0.0025,
		"chatgpt-image-latest": 0.002,
		"gpt-image-1-mini":     0.00025,
	},
	"images-tokens-output": {
		"gpt-image-1.5":        0.032,
		"gpt-image-1":          0.040,
		"chatgpt-image-latest": 0.032,
		"gpt-image-1-mini":     0.008,
	},
}

var imageModelsWithTokensCost = map[string]interface{}{}

func init() {
	for model := range OpenAiPerThousandTokenCost["images-tokens-input"] {
		imageModelsWithTokensCost[model] = struct{}{}
	}

	for _, tool := range AllowedTools {
		AllowedToolsSet[tool] = struct{}{}
	}
}

var OpenAiPerThousandCallsToolCost = map[string]float64{
	"web_search":                   10.0,
	"web_search_preview":           25.0,
	"web_search_preview_reasoning": 10.0,
	"file_search":                  2.5,
}

var OpenAiCodeInterpreterContainerCost = map[string]float64{
	"1g":  0.03,
	"4g":  0.12,
	"16g": 0.48,
	"64g": 1.92,
}

var AllowedTools = []string{
	"web_search",
	"web_search_preview",
	"web_search_preview_2025_03_11",

	"code_interpreter",

	"file_search",
	"function",
	"computer_use_preview",
	"computer",
	"computer_use",
	"exec_command",
	"shell",
	"apply_patch",
	"filesystem",
	"patch",
	"namespace",
	"custom",
	"custom_code",
	"mcp",
	"tool_search",
	"image_generation",
	"skills",
}

var AllowedToolsSet = map[string]struct{}{}

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
	totalTokens := int64(promptTks + completionTks)
	model = provider.ModelWithContextLength(model, totalTokens)
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

// EstimateChatCompletionStreamCostWithTokenCounts estimates the cost of a
// streamed chat completion. promptTks must be the already-counted prompt
// token count so that the long/short context tier is picked using
// promptTks+completionTks, matching the tiering used by EstimateTotalCost
// for non-streaming requests.
func (ce *CostEstimator) EstimateChatCompletionStreamCostWithTokenCounts(model string, promptTks int, content string) (int, float64, float64, error) {
	if len(model) == 0 {
		return 0, 0, 0, errors.New("model is not provided")
	}

	tks, err := ce.tc.Count(model, content)
	if err != nil {
		return 0, 0, 0, err
	}

	tieredModel := provider.ModelWithContextLength(model, int64(promptTks+tks))

	promptCost, err := ce.EstimatePromptCost(tieredModel, promptTks)
	if err != nil {
		return 0, 0, 0, err
	}

	completionCost, err := ce.EstimateCompletionCost(tieredModel, tks)
	if err != nil {
		return 0, 0, 0, err
	}

	return tks, promptCost, completionCost, nil
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

func (ce *CostEstimator) estimateImageByMetadata(model string, metadata *ImageResponseMetadata) (float64, error) {
	if metadata == nil {
		return 0, errors.New("metadata is nil")
	}
	if _, ok := imageModelsWithTokensCost[model]; !ok {
		return 0, errors.New("model is not present in the images tokens cost map")
	}
	var totalCost float64

	textInputTokens := metadata.Usage.InputTokensDetails.TextTokens
	textInputCostMap, ok := ce.tokenCostMap["prompt"]
	if !ok {
		return 0, errors.New("images input tokens cost map is not provided")
	}
	textInputCost, _ := textInputCostMap[model]
	totalCost += (float64(textInputTokens) / 1000) * textInputCost

	imageInputTokens := metadata.Usage.InputTokensDetails.ImageTokens
	imageInputCostMap, ok := ce.tokenCostMap["images-tokens-input"]
	if !ok {
		return 0, errors.New("images input tokens cost map is not provided")
	}
	imageInputCost, _ := imageInputCostMap[model]
	totalCost += (float64(imageInputTokens) / 1000) * imageInputCost

	outputTokens := metadata.Usage.OutputTokens
	imageOutputCostMap, ok := ce.tokenCostMap["images-tokens-output"]
	if !ok {
		return 0, errors.New("images output tokens cost map is not provided")
	}
	imageOutputCost, _ := imageOutputCostMap[model]
	totalCost += (float64(outputTokens) / 1000) * imageOutputCost

	return totalCost, nil
}

func (ce *CostEstimator) EstimateImagesCost(model, quality, resolution string, metadata *ImageResponseMetadata) (float64, error) {
	mCost, err := ce.estimateImageByMetadata(model, metadata)
	if err == nil {
		return mCost, nil
	}

	costMap, ok := ce.tokenCostMap["images"]
	if !ok {
		return 0, errors.New("images cost map is not provided")
	}

	// Try to prepare normalized model key with resolution/quality
	simpleRes, resErr := convertResToSimple(resolution)
	var normalizedModel string
	var prepareErr error

	if resErr == nil {
		switch model {
		case "dall-e-2":
			normalizedModel, prepareErr = prepareDallE2Model(simpleRes, model)
		case "dall-e-3":
			normalizedModel, prepareErr = prepareDallE3Model(quality, simpleRes, model)
		case "gpt-image-1", "gpt-image-1.5", "chatgpt-image-latest", "gpt-image-1-mini":
			normalizedModel, prepareErr = prepareGptImageModel(quality, simpleRes, model)
		default:
			// Unknown model, will try model-only fallback
			prepareErr = errors.New("unknown model type")
		}

		// If normalization succeeded, try the normalized key first
		if prepareErr == nil {
			if cost, ok := costMap[normalizedModel]; ok {
				return cost, nil
			}
		}
	}

	// Fall back to model-only lookup
	if cost, ok := costMap[model]; ok {
		return cost, nil
	}

	return 0, errors.New("model (with or without quality/resolution) is not present in the images cost map")
}

var allowedDallE2Resolutions = []string{"256", "512", "1024"}
var allowedDallE3Resolutions = []string{"1024", "1792"}
var allowedDallE3Qualities = []string{"standart", "hd"}

func convertResToSimple(resolution string) (string, error) {
	if resolution == "" {
		return "", nil
	}
	if strings.Contains(resolution, "1792") {
		return "1792", nil
	}
	if strings.Contains(resolution, "1536") {
		return "1536", nil
	}
	if strings.Contains(resolution, "1024") {
		return "1024", nil
	}
	if strings.Contains(resolution, "512") {
		return "512", nil
	}
	if strings.Contains(resolution, "256") {
		return "256", nil
	}
	return "", errors.New("resolution is not valid")
}

func prepareDallE2Model(resolution, model string) (string, error) {
	if resolution == "" {
		return model, nil
	}
	if slices.Contains(allowedDallE2Resolutions, resolution) {
		return fmt.Sprintf("%s-%s", model, resolution), nil
	}
	return "", errors.New("resolution is not valid")
}

func prepareDallE3Model(quality, resolution, model string) (string, error) {
	preparedQuality, err := prepareDallE3Quality(quality)
	if err != nil {
		return "", err
	}
	if resolution == "" && quality == "" {
		return model, nil
	}
	if resolution == "" {
		return fmt.Sprintf("%s-%s-%s", model, "1024", preparedQuality), nil
	}
	if slices.Contains(allowedDallE3Resolutions, resolution) {
		return fmt.Sprintf("%s-%s-%s", model, resolution, preparedQuality), nil
	}
	return "", errors.New("resolution is not valid")
}

func prepareDallE3Quality(quality string) (string, error) {
	if quality != "" && !slices.Contains(allowedDallE3Qualities, quality) {
		return "", errors.New("quality is not valid")
	}
	if quality == "" {
		return "standart", nil
	}
	return quality, nil
}

var allowedGptImageResolutions = []string{"1024", "1536", "auto"}
var allowedGptImageQualities = []string{"low", "medium", "high", "auto"}

func prepareGptImageModel(quality, resolution, model string) (string, error) {
	preparedQuality, err := prepareGptImageQuality(quality)
	if err != nil {
		return "", err
	}
	simpleRes, err := convertResToSimple(resolution)
	if err != nil {
		return "", err
	}
	preparedResolution, err := prepareGptImageResolution(simpleRes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s", model, preparedResolution, preparedQuality), nil
}

func prepareGptImageResolution(resolution string) (string, error) {
	if resolution != "" && !slices.Contains(allowedGptImageResolutions, resolution) {
		return "", errors.New("resolution is not valid")
	}
	if resolution == "" || resolution == "auto" {
		return "1536", nil
	}
	return resolution, nil
}

func prepareGptImageQuality(quality string) (string, error) {
	if quality != "" && !slices.Contains(allowedGptImageQualities, quality) {
		return "", errors.New("quality is not valid")
	}
	if quality == "" || quality == "auto" {
		return "high", nil
	}
	return quality, nil
}

func (ce *CostEstimator) EstimateTranscriptionCost(secs float64, model string, usage *TranscriptionResponseUsage) (float64, error) {
	if usage != nil {
		inputTokens := usage.InputTokens
		costMap, ok := ce.tokenCostMap["transcription-input"]
		if !ok {
			return 0, errors.New("transcription input token cost map is not provided")
		}
		inputCost, ok := costMap[model]
		if !ok {
			return 0, errors.New("model is not present in the transcription input token cost map")
		}

		outputTokens := usage.OutputTokens
		costMap, ok = ce.tokenCostMap["transcription-output"]
		if !ok {
			return 0, errors.New("transcription output token cost map is not provided")
		}
		outputCost, ok := costMap[model]
		if !ok {
			return 0, errors.New("model is not present in the transcription output token cost map")
		}

		return (float64(inputTokens)/1000)*inputCost + (float64(outputTokens)/1000)*outputCost, nil
	}
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

func (ce *CostEstimator) EstimateResponseApiTotalCost(model string, usage responsesOpenai.ResponseUsage) (float64, error) {
	if len(model) == 0 {
		return 0, errors.New("model is not provided")
	}

	inputTokens := usage.InputTokens
	cachedInputTokens := usage.InputTokensDetails.CachedTokens
	outputTokens := usage.OutputTokens

	totalTokens := inputTokens + cachedInputTokens + outputTokens
	model = provider.ModelWithContextLength(model, totalTokens)

	cachedInputCost, err := ce.estimateResponseApiTokensCost("cached-prompt", model, cachedInputTokens)
	if err != nil {
		cachedInputTokens = 0.0
	}
	inputCost, err := ce.estimateResponseApiTokensCost("prompt", model, inputTokens-cachedInputTokens)
	if err != nil {
		return 0.0, err
	}

	outputCost, err := ce.estimateResponseApiTokensCost("completion", model, outputTokens)

	return inputCost + cachedInputCost + outputCost, err
}

func (ce *CostEstimator) EstimateResponseApiToolCallsCost(tools []responsesOpenai.ToolUnion, model string) (float64, error) {
	if len(tools) == 0 {
		return 0, nil
	}
	totalCost := 0.0
	for _, tool := range tools {
		toolType := tool.Type
		cost, ok := OpenAiPerThousandCallsToolCost[extendedToolType(toolType, model)]
		if !ok {
			// The permission check is performed in the middleware. The additional cost is taken from OpenAiPerThousandCallsToolCost
			continue
		}
		totalCost += cost
	}
	return totalCost / 1000, nil
}

func (ce *CostEstimator) EstimateResponseApiToolCreateContainerCost(req *ResponseRequest) (float64, error) {
	if req == nil {
		return 0, nil
	}
	totalCost := 0.0
	for _, tool := range req.Tools {
		c := tool.GetContainerAsResponseRequestToolContainer()
		if c == nil {
			continue
		}
		limit := c.GetMemoryLimit()
		cost, ok := OpenAiCodeInterpreterContainerCost[limit]
		if !ok {
			return 0, fmt.Errorf("container with memory limit %s is not present in the code interpreter container cost map", limit)
		}
		totalCost += cost
	}
	return totalCost, nil
}

func (ce *CostEstimator) EstimateVideoCost(metadata *VideoResponseMetadata) (float64, error) {
	if metadata == nil {
		return 0, errors.New("metadata is nil")
	}
	costMap, ok := ce.tokenCostMap["video"]
	if !ok {
		return 0, errors.New("video cost map is not provided")
	}
	model := metadata.Model

	// Validate and get seconds as float
	seconds, err := metadata.GetSecondsAsFloat()
	if err != nil {
		return 0, fmt.Errorf("failed to parse seconds field: %w", err)
	}

	// Try model-size lookup first if size is present and can be normalized
	size, err := normalizedVideoSize(metadata.Size)
	if err == nil && size != "" {
		costKey := fmt.Sprintf("%s-%s", model, size)
		if cost, ok := costMap[costKey]; ok {
			return cost * seconds, nil
		}
	}

	// Fall back to model-only lookup
	if cost, ok := costMap[model]; ok {
		return cost * seconds, nil
	}

	return 0, errors.New("model (with or without size) is not present in the video cost map")
}

func normalizedVideoSize(size string) (string, error) {
	if size == "" {
		return "", nil
	}
	switch size {
	case "720x1280", "1280x720":
		return "720", nil
	case "1024x1792", "1792x1024":
		return "1024", nil
	case "1080x1920", "1920x1080":
		return "1080", nil
	default:
		return "", errors.New("size is not valid")
	}
}

var reasoningModelPrefix = []string{"gpt-5", "o1", "o2", "o3"}

func extendedToolType(toolType, model string) string {
	if toolType != "web_search_preview" {
		return toolType
	}
	if slices.ContainsFunc(reasoningModelPrefix, func(s string) bool { return strings.HasPrefix(model, s) }) {
		return "web_search_preview_reasoning"
	}
	return toolType
}

func (ce *CostEstimator) estimateResponseApiTokensCost(costMapKey, model string, tks int64) (float64, error) {
	costMap, ok := ce.tokenCostMap[costMapKey]
	if !ok {
		return 0, errors.New("cost map is not provided")
	}
	cost, ok := costMap[model]
	if !ok {
		return 0, fmt.Errorf("%s is not present in the cost map provided", model)
	}
	tksInFloat := float64(tks)
	return tksInFloat / 1000 * cost, nil
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
