package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logRetrieveMessageFileRequest(log *zap.Logger, prod bool, tid, mid, fid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String("thread_id", tid),
			zap.String("message_id", mid),
			zap.String("file_id", fid),
		}

		log.Info("openai retrieve message file request", fields...)
	}
}

func logRetrieveMessageFileResponse(log *zap.Logger, data []byte, prod bool) {
	mf := &goopenai.MessageFile{}
	err := json.Unmarshal(data, mf)
	if err != nil {
		logError(log, "error when unmarshalling message file response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("id", mf.ID),
			zap.String("object", mf.Object),
			zap.Int("created_at", mf.CreatedAt),
			zap.String("message_id", mf.MessageID),
		}

		log.Info("openai message file response", fields...)
	}
}

func logListMessageFilesRequest(log *zap.Logger, prod bool, tid, mid string, params map[string]string) {
	if prod {
		fields := []zapcore.Field{
			zap.String("thread_id", tid),
			zap.String("message_id", mid),
		}

		if v, ok := params["limit"]; ok {
			fields = append(fields, zap.String("limit", v))
		}

		if v, ok := params["order"]; ok {
			fields = append(fields, zap.String("order", v))
		}

		if v, ok := params["after"]; ok {
			fields = append(fields, zap.String("after", v))
		}

		if v, ok := params["before"]; ok {
			fields = append(fields, zap.String("before", v))
		}

		log.Info("openai list message files request", fields...)
	}
}

func logListMessageFilesResponse(log *zap.Logger, data []byte, prod bool) {
	files := &goopenai.MessageFilesList{}
	err := json.Unmarshal(data, files)
	if err != nil {
		logError(log, "error when unmarshalling list message files response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.Any("message_files", files.MessageFiles),
		}

		log.Info("openai list message files response", fields...)
	}
}
