package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logListFilesRequest(log *zap.Logger, prod bool, cid string, params map[string]string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
		}

		if v, ok := params["purpose"]; ok {
			fields = append(fields, zap.String("purpose", v))
		}

		log.Info("openai list files request", fields...)
	}
}

func logListFilesResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	files := &goopenai.FilesList{}
	err := json.Unmarshal(data, files)
	if err != nil {
		logError(log, "error when unmarshalling list files response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.Any("files", files.Files),
		}

		log.Info("openai list files response", fields...)
	}
}

func logRetrieveFileRequest(log *zap.Logger, prod bool, cid, fid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("file_id", fid),
		}

		log.Info("openai retrieve file request", fields...)
	}
}

func logRetrieveFileResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	file := &goopenai.File{}
	err := json.Unmarshal(data, file)
	if err != nil {
		logError(log, "error when unmarshalling retrieve file response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("id", file.ID),
			zap.Int("bytes", file.Bytes),
			zap.Int64("createdAt", file.CreatedAt),
			zap.String("filename", file.FileName),
			zap.String("object", file.Object),
			zap.String("purpose", file.Purpose),
			zap.String("status", file.Status),
			zap.String("status_details", file.StatusDetails),
		}

		log.Info("openai retrieve file request", fields...)
	}
}

func logDeleteFileRequest(log *zap.Logger, prod bool, cid, fid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("file_id", fid),
		}

		log.Info("openai delete file request", fields...)
	}
}

func logDeleteFileResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	dr := &DeletionResponse{}
	err := json.Unmarshal(data, dr)
	if err != nil {
		logError(log, "error when unmarshalling delete file response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("id", dr.Id),
			zap.String("object", dr.Object),
			zap.Bool("deleted", dr.Deleted),
		}

		log.Info("openai delete file response", fields...)
	}
}

func logRetrieveFileContentRequest(log *zap.Logger, data []byte, prod bool, cid, fid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("file_id", fid),
		}

		log.Info("openai retrieve file content request", fields...)
	}
}

func logRetrieveFileContentResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
		}

		log.Info("openai retrieve file content response", fields...)
	}
}

func logUploadFileRequest(log *zap.Logger, data []byte, prod bool, cid, purpose string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("purpose", purpose),
		}

		log.Info("openai upload file request", fields...)
	}
}

func logUploadFileResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	file := &goopenai.File{}
	err := json.Unmarshal(data, file)
	if err != nil {
		logError(log, "error when unmarshalling retrieve file response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("id", file.ID),
			zap.Int("bytes", file.Bytes),
			zap.Int64("createdAt", file.CreatedAt),
			zap.String("filename", file.FileName),
			zap.String("object", file.Object),
			zap.String("purpose", file.Purpose),
			zap.String("status", file.Status),
			zap.String("status_details", file.StatusDetails),
		}

		log.Info("openai retrieve file request", fields...)
	}
}
