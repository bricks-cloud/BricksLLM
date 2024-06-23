package vllm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/util"
)

type CostEstimator struct {
	tc tokenCounter
}

type tokenCounter interface {
	Count(model string, input string) int
}

func NewCostEstimator(tc tokenCounter) *CostEstimator {
	return &CostEstimator{
		tc: tc,
	}
}

func (ce *CostEstimator) EstimateCompletionPromptToken(r *CompletionRequest) int {
	content, err := util.ConvertAnyToStr(r.Prompt)
	if err != nil {
		return 0
	}

	if len(content) == 0 {
		return 0
	}

	return ce.EstimateContentTokenCounts(r.Model, content)
}

func (ce *CostEstimator) EstimateChatCompletionPromptToken(r *ChatRequest) int {
	return countTotalTokens(r.Model, r, ce.tc)
}

func (ce *CostEstimator) EstimateContentTokenCounts(model string, content string) int {
	return ce.tc.Count(model, content)
}

func countFunctionTokens(model string, r *ChatRequest, tc tokenCounter) int {
	if len(r.Functions) == 0 {
		return 0
	}

	defs := formatFunctionDefinitions(r)
	tks := tc.Count(model, defs)

	tks += 9
	return tks
}

func formatFunctionDefinitions(r *ChatRequest) string {
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

func countMessageTokens(model string, r *ChatRequest, tc tokenCounter) int {
	if len(r.Messages) == 0 {
		return 0
	}

	result := 0
	padded := false

	for _, msg := range r.Messages {
		content := msg.Content
		if msg.Role == "system" && !padded {
			content += "\n"
			padded = true
		}

		contentTks := tc.Count(model, content)
		roleTks := tc.Count(model, msg.Role)
		nameTks := tc.Count(model, msg.Name)
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

	return result
}

func countTotalTokens(model string, r *ChatRequest, tc tokenCounter) int {
	if r == nil {
		return 0
	}

	tks := 3

	ftks := countFunctionTokens(model, r, tc)

	mtks := countMessageTokens(model, r, tc)

	systemExists := false
	for _, msg := range r.Messages {
		if msg.Role == "system" {
			systemExists = true
		}

	}

	if len(r.Functions) != 0 && systemExists {
		tks -= 4
	}

	return tks + ftks + mtks
}
