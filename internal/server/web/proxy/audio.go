package proxy

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SpeechRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

func logCreateSpeechRequest(log *zap.Logger, sr *SpeechRequest, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", sr.Model),
			zap.String("voice", sr.Voice),
		}

		if !private {
			fields = append(fields, zap.String("input", sr.Input))
		}

		if len(sr.ResponseFormat) != 0 {
			fields = append(fields, zap.String("response_format", sr.ResponseFormat))
		}

		if sr.Speed != 0 {
			fields = append(fields, zap.Float64("speed", sr.Speed))
		}

		log.Info("openai create speech request", fields...)
	}
}

func logCreateTranscriptionRequest(log *zap.Logger, model, language, prompt, responseFormat string, temperature float64, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", model),
		}

		if !private && len(prompt) != 0 {
			fields = append(fields, zap.String("prompt", prompt))
		}

		if len(language) != 0 {
			fields = append(fields, zap.String("language", language))
		}

		if len(responseFormat) != 0 {
			fields = append(fields, zap.String("response_format", responseFormat))
		}

		if temperature != 0 {
			fields = append(fields, zap.Float64("temperature", temperature))
		}

		log.Info("openai create transcription request", fields...)
	}
}

func logCreateTranslationRequest(log *zap.Logger, model, prompt, responseFormat string, temperature float64, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", model),
		}

		if !private && len(prompt) == 0 {
			fields = append(fields, zap.String("prompt", prompt))
		}

		if len(responseFormat) != 0 {
			fields = append(fields, zap.String("response_format", responseFormat))
		}

		log.Info("openai create translation request", fields...)
	}
}
