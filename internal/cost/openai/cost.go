package openai

import "strings"

func EstimateCost(model string, promptTokens int, completionTokens int) float64 {
	if strings.HasPrefix(model, "gpt-4") {
		if strings.HasPrefix(model, "gpt-4-32k") {
			return float64(promptTokens)*0.06 + float64(completionTokens)*0.12
		}

		return float64(promptTokens)*0.03 + float64(completionTokens)*0.06
	}

	if strings.HasPrefix(model, "gpt-3.5-turbo") {
		if strings.HasPrefix(model, "gpt-3.5-turbo-16k") {
			return float64(promptTokens)*0.003 + float64(completionTokens)*0.004
		}

		return float64(promptTokens)*0.0015 + float64(completionTokens)*0.002
	}

	return 0
}
