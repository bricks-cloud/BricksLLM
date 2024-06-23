package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ModerationRequest struct {
	Input any    `json:"input"`
	Model string `json:"model"`
}

func logCreateModerationRequest(log *zap.Logger, data []byte, prod, private bool) {
	mr := &ModerationRequest{}
	err := json.Unmarshal(data, mr)
	if err != nil {
		logError(log, "error when unmarshalling create moderation request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{}

		if !private {
			fields = append(fields, zap.Any("input", mr.Input))
		}

		log.Info("openai create moderation request", fields...)
	}
}

func logCreateModerationResponse(log *zap.Logger, data []byte, prod bool) {
	mr := &goopenai.ModerationResponse{}
	err := json.Unmarshal(data, mr)
	if err != nil {
		logError(log, "error when unmarshalling create moderation response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("id", mr.ID),
			zap.String("model", mr.Model),
			zap.Any("results", mr.Results),
		}

		log.Info("openai create moderation response", fields...)
	}
}
