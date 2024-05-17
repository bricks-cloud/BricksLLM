package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCreateAssistantFileRequest(log *zap.Logger, data []byte, prod bool, cid, aid string) {
	afr := &goopenai.AssistantFileRequest{}
	err := json.Unmarshal(data, afr)
	if err != nil {
		logError(log, "error when unmarshalling create assistant file request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", aid),
			zap.String("file_id", afr.FileID),
		}

		log.Info("openai create assistant file request", fields...)
	}
}

func logAssistantFileResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	af := &goopenai.AssistantFile{}
	err := json.Unmarshal(data, af)
	if err != nil {
		logError(log, "error when unmarshalling assistant file request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", af.AssistantID),
			zap.String("id", af.ID),
			zap.String("object", af.Object),
			zap.Int64("created_at", af.CreatedAt),
		}

		log.Info("openai create assistant file response", fields...)
	}
}

func logRetrieveAssistantFileRequest(log *zap.Logger, prod bool, cid, fid, aid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", aid),
			zap.String("file_id", fid),
		}

		log.Info("openai create assistant file request", fields...)
	}
}

func logDeleteAssistantFileRequest(log *zap.Logger, prod bool, cid, fid, aid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", aid),
			zap.String("file_id", fid),
		}

		log.Info("openai delete assistant file request", fields...)
	}
}

func logDeleteAssistantFileResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	dr := &goopenai.AssistantDeleteResponse{}
	err := json.Unmarshal(data, dr)
	if err != nil {
		logError(log, "error when unmarshalling delete assistant file response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("id", dr.ID),
			zap.String("object", dr.Object),
			zap.Bool("deleted", dr.Deleted),
		}

		log.Info("openai delete assistant file response", fields...)
	}
}

func logListAssistantFilesRequest(log *zap.Logger, prod bool, cid, aid string, params map[string]string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.String("assistant_id", aid),
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

		log.Info("openai list assistant files request", fields...)
	}
}

func logListAssistantFilesResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	files := &goopenai.AssistantFilesList{}
	err := json.Unmarshal(data, files)
	if err != nil {
		logError(log, "error when unmarshalling list assistant files response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(logFiledNameCorrelationId, cid),
			zap.Any("assistant_files", files.AssistantFiles),
		}

		log.Info("openai list assistant files response", fields...)
	}
}
