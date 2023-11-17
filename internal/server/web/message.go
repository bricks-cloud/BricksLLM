package web

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCreateMessageRequest(log *zap.Logger, data []byte, prod, private bool, cid string) {
	mr := &goopenai.MessageRequest{}
	err := json.Unmarshal(data, mr)
	if err != nil {
		logError(log, "error when unmarshalling create message request", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("role", mr.Role),
			zap.Any("file_ids", mr.FileIds),
			zap.Any("metadata", mr.Metadata),
		}

		if !private && len(mr.Content) != 0 {
			fields = append(fields, zap.String("content", mr.Content))
		}

		log.Info("openai create message request", fields...)
	}
}

func logMessageResponse(log *zap.Logger, data []byte, prod, private bool, cid string) {
	m := &goopenai.Message{}
	err := json.Unmarshal(data, m)
	if err != nil {
		logError(log, "error when unmarshalling message response", prod, cid, err)
		return
	}

	if private {
		for _, mc := range m.Content {
			mc.Text = nil
		}
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("id", m.ID),
			zap.String("object", m.Object),
			zap.Int("created_at", m.CreatedAt),
			zap.String("role", m.Role),
			zap.Any("content", m.Content),
			zap.Any("file_ids", m.FileIds),
			zap.Any("assistant_id", m.AssistantID),
			zap.Any("run_id", m.RunID),
			zap.Any("metadata", m.Metadata),
		}

		log.Info("openai message response", fields...)
	}
}

func logRetrieveMessageRequest(log *zap.Logger, prod bool, cid, mid, tid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("thread_id", tid),
			zap.String("message_id", mid),
		}

		log.Info("openai retrieve message request", fields...)
	}
}

func logModifyMessageRequest(log *zap.Logger, data []byte, prod, private bool, cid, tid, mid string) {
	mr := &goopenai.MessageRequest{}
	err := json.Unmarshal(data, mr)
	if err != nil {
		logError(log, "error when unmarshalling modify message request", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("thread_id", tid),
			zap.String("message_id", mid),
			zap.Any("metadata", mr.Metadata),
		}

		if !private && len(mr.Content) != 0 {
			fields = append(fields, zap.String("content", mr.Content))
		}

		log.Info("openai modify message request", fields...)
	}
}

func logListMessagesRequest(log *zap.Logger, data []byte, prod bool, cid, tid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("thread_id", tid),
		}

		log.Info("openai list messages request", fields...)
	}
}

func logListMessagesResponse(log *zap.Logger, data []byte, prod, private bool, cid string) {
	ms := &goopenai.MessagesList{}
	err := json.Unmarshal(data, ms)
	if err != nil {
		logError(log, "error when unmarshalling list messages response", prod, cid, err)
		return
	}

	for _, m := range ms.Messages {
		if private {
			for _, mc := range m.Content {
				mc.Text = nil
			}
		}
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.Any("messages", ms.Messages),
		}

		log.Info("openai list messages response", fields...)
	}
}

func logRetrieveMessageFileRequest(log *zap.Logger, prod bool, cid, tid, mid, fid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("thread_id", tid),
			zap.String("message_id", mid),
			zap.String("file_id", fid),
		}

		log.Info("openai retrieve message file request", fields...)
	}
}

func logRetrieveMessageFileResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	mf := &goopenai.MessageFile{}
	err := json.Unmarshal(data, mf)
	if err != nil {
		logError(log, "error when unmarshalling message file response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("id", mf.ID),
			zap.String("object", mf.Object),
			zap.Int("created_at", mf.CreatedAt),
			zap.String("message_id", mf.MessageID),
		}

		log.Info("openai message file response", fields...)
	}
}

func logListMessageFilesRequest(log *zap.Logger, data []byte, prod bool, cid, tid, mid string, params map[string]string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
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

func logListMessageFilesResponse(log *zap.Logger, data []byte, prod bool, cid string) {
	files := &goopenai.MessageFilesList{}
	err := json.Unmarshal(data, files)
	if err != nil {
		logError(log, "error when unmarshalling list message files response", prod, cid, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.Any("message_files", files.MessageFiles),
		}

		log.Info("openai list message files response", fields...)
	}
}
