package proxy

import (
	"encoding/json"

	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCompletionRequest(log *zap.Logger, data []byte, prod, private bool, cid string) {
	cr := &anthropic.CompletionRequest{}
	err := json.Unmarshal(data, cr)
	if err != nil {
		logError(log, "error when unmarshalling anthropic completion request", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", cr.Model),
			zap.Int("max_tokens_to_sample", cr.MaxTokensToSample),
			zap.Any("stop_sequnces", cr.StopSequences),
			zap.Float32("temperature", cr.Temperature),
			zap.Int("top_p", cr.TopP),
			zap.Int("top_k", cr.TopK),
			zap.Bool("stream", cr.Stream),
		}

		if cr.Metadata != nil {
			fields = append(fields, zap.Any("metadata", cr.Metadata))
		}

		if !private {
			fields = append(fields, zap.String("prompt", cr.Prompt))
		}

		log.Info("anthropic completion request", fields...)
	}
}

func logCompletionResponse(log *zap.Logger, data []byte, prod, private bool, cid string) {
	cr := &anthropic.CompletionResponse{}
	err := json.Unmarshal(data, cr)
	if err != nil {
		logError(log, "error when unmarshalling anthropic completion response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("stop_reason", cr.StopReason),
			zap.String("model", cr.Model),
		}

		if !private {
			fields = append(fields, zap.String("completion", cr.Completion))
		}

		log.Info("anthropic completion response", fields...)
	}
}
