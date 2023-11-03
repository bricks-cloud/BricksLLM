package event

import "github.com/bricks-cloud/bricksllm/internal/provider"

type Event struct {
	Id                   string
	CreatedAt            int64
	Tags                 []string
	KeyId                string
	CostInUsd            float64
	Provider             provider.Provider
	Model                string
	Status               int
	PromptTokenCount     int
	CompletionTokenCount int
	LatencyInMs          int
}
