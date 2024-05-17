package proxy

import (
	"encoding/json"

	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	goopenai "github.com/sashabaranov/go-openai"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCreateThreadRequest(log *zap.Logger, data []byte, prod, private bool, cid string) {
	tr := &openai.ThreadRequest{}
	err := json.Unmarshal(data, tr)
	if err != nil {
		logError(log, "error when unmarshalling create thread request", prod, err)
		return
	}

	if private {
		for _, m := range tr.Messages {
			m.Content = ""
		}
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.Any("metadata", tr.Metadata),
			zap.Any("messages", tr.Messages),
		}

		log.Info("openai create thread request", fields...)
	}
}

func logThreadResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	t := &goopenai.Thread{}
	err := json.Unmarshal(data, t)
	if err != nil {
		logError(log, "error when unmarshalling thread response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", t.ID),
			zap.String("object", t.Object),
			zap.Int64("created_at", t.CreatedAt),
			zap.Any("metadata", t.Metadata),
		}

		log.Info("openai create thread response", fields...)
	}
}

func logRetrieveThreadRequest(log *zap.Logger, prod bool, cid, tid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", tid),
		}

		log.Info("openai retrieve thread request", fields...)
	}
}

func logModifyThreadRequest(log *zap.Logger, data []byte, prod bool, cid, tid string) {
	tr := &goopenai.ThreadRequest{}
	err := json.Unmarshal(data, tr)
	if err != nil {
		logError(log, "error when unmarshalling modify thread request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", tid),
			zap.Any("metadata", tr.Metadata),
		}

		log.Info("openai modify thread request", fields...)
	}
}

func logDeleteThreadRequest(log *zap.Logger, prod bool, cid, tid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", tid),
		}

		log.Info("openai delete thread request", fields...)
	}
}

func logDeleteThreadResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	tdr := &goopenai.ThreadDeleteResponse{}
	err := json.Unmarshal(data, tdr)
	if err != nil {
		logError(log, "error when unmarshalling thread deletion response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", tdr.ID),
			zap.String("object", tdr.Object),
			zap.Bool("deleted", tdr.Deleted),
		}

		log.Info("openai thread deletion response", fields...)
	}
}
