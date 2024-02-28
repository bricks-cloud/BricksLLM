package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCreateImageRequest(log *zap.Logger, ir *goopenai.ImageRequest, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", ir.Model),
			zap.Int("n", ir.N),
			zap.String("quality", ir.Quality),
			zap.String("size", ir.Size),
			zap.String("style", ir.Style),
			zap.String("response_format", ir.ResponseFormat),
			zap.String("user", ir.User),
		}

		if !private && len(ir.Prompt) != 0 {
			fields = append(fields, zap.String("prompt", ir.Prompt))
		}

		log.Info("openai create image request", fields...)
	}
}

func logEditImageRequest(log *zap.Logger, prompt, model string, n int, size, responseFormat, user string, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
		}

		if !private && len(prompt) != 0 {
			fields = append(fields, zap.String("prompt", prompt))
		}

		if len(model) != 0 {
			fields = append(fields, zap.String("prompt", model))
		}

		if n != 0 {
			fields = append(fields, zap.Int("n", n))
		}

		if len(size) != 0 {
			fields = append(fields, zap.String("size", size))
		}

		if len(responseFormat) != 0 {
			fields = append(fields, zap.String("response_format", responseFormat))
		}

		if len(user) != 0 {
			fields = append(fields, zap.String("user", user))
		}

		log.Info("openai edit image request", fields...)
	}
}

func logImageVariationsRequest(log *zap.Logger, model string, n int, size, responseFormat, user string, prod bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
		}

		if len(model) != 0 {
			fields = append(fields, zap.String("model", model))
		}

		if n != 0 {
			fields = append(fields, zap.Int("n", n))
		}

		if len(size) != 0 {
			fields = append(fields, zap.String("size", size))
		}

		if len(responseFormat) != 0 {
			fields = append(fields, zap.String("response_format", user))
		}

		if len(user) != 0 {
			fields = append(fields, zap.String("user", user))
		}

		log.Info("openai image variations request", fields...)
	}
}

func logImageResponse(log *zap.Logger, data []byte, prod, private bool, cid string) {
	ir := &goopenai.ImageResponse{}
	err := json.Unmarshal(data, ir)
	if err != nil {
		logError(log, "error when unmarshalling image response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.Int64("created", ir.Created),
		}

		if !private && len(ir.Data) != 0 {
			fields = append(fields, zap.Any("data", ir.Data))
		}

		log.Info("openai image response", fields...)
	}
}
