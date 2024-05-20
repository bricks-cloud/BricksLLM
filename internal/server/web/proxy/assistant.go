package proxy

import (
	"encoding/json"

	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logCreateAssistantRequest(log *zap.Logger, data []byte, prod, private bool) {
	ar := &goopenai.AssistantRequest{}
	err := json.Unmarshal(data, ar)
	if err != nil {
		logError(log, "error when unmarshalling assistant request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("model", ar.Model),
			zap.Any("tools", ar.Tools),
			zap.Any("file_ids", ar.FileIDs),
			zap.Any("metadata", ar.Metadata),
			zap.Stringp("name", ar.Name),
			zap.Stringp("description", ar.Description),
		}

		if !private && ar.Instructions != nil {
			fields = append(fields, zap.String("instructions", *ar.Instructions))
		}

		log.Info("openai create assistant request", fields...)
	}
}

func logAssistantResponse(log *zap.Logger, data []byte, prod, private bool) {
	a := &goopenai.Assistant{}
	err := json.Unmarshal(data, a)
	if err != nil {
		logError(log, "error when unmarshalling assistant response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("object", a.Object),
			zap.Int64("created_at", a.CreatedAt),
			zap.String("Model", a.Model),
			zap.Any("tools", a.Tools),
			zap.Stringp("name", a.Name),
			zap.Stringp("description", a.Description),
		}

		if !private && a.Instructions != nil {
			fields = append(fields, zap.String("instructions", *a.Instructions))
		}

		log.Info("openai create assistant response", fields...)
	}
}

func logRetrieveAssistantRequest(log *zap.Logger, prod bool, assistantId string) {
	if prod {
		fields := []zapcore.Field{
			zap.String("id", assistantId),
		}

		log.Info("openai retrieve assistant request", fields...)
	}
}

func logModifyAssistantRequest(log *zap.Logger, data []byte, prod, private bool, assistantId string) {
	ar := &goopenai.AssistantRequest{}
	err := json.Unmarshal(data, ar)
	if err != nil {
		logError(log, "error when unmarshalling modifying assistant request", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("id", assistantId),
			zap.String("model", ar.Model),
			zap.Any("tools", ar.Tools),
			zap.Any("file_ids", ar.FileIDs),
			zap.Any("metadata", ar.Metadata),
			zap.Stringp("name", ar.Name),
			zap.Stringp("description", ar.Description),
		}

		if !private && ar.Instructions != nil {
			fields = append(fields, zap.String("instructions", *ar.Instructions))
		}

		log.Info("openai modify assistant request", fields...)
	}
}

func logDeleteAssistantRequest(log *zap.Logger, prod bool, assistantId string) {
	if prod {
		fields := []zapcore.Field{
			zap.String("id", assistantId),
		}

		log.Info("openai delete assistant request", fields...)
	}
}

func logDeleteAssistantResponse(log *zap.Logger, data []byte, prod bool) {
	adr := &goopenai.AssistantDeleteResponse{}
	err := json.Unmarshal(data, adr)
	if err != nil {
		logError(log, "error when unmarshalling assistant deletion response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{
			zap.String("id", adr.ID),
			zap.String("object", adr.Object),
			zap.Bool("deleted", adr.Deleted),
		}

		log.Info("openai assistant deletion response", fields...)
	}
}

func logListAssistantsRequest(log *zap.Logger, prod bool) {
	if prod {
		fields := []zapcore.Field{}

		log.Info("openai list assistants request", fields...)
	}
}

func logListAssistantsResponse(log *zap.Logger, data []byte, prod, private bool) {
	assistants := &goopenai.AssistantsList{}
	err := json.Unmarshal(data, assistants)
	if err != nil {
		logError(log, "error when unmarshalling list assistants response", prod, err)
		return
	}

	for _, assistant := range assistants.Assistants {
		if private {
			assistant.Instructions = nil
		}
	}

	if prod {
		fields := []zapcore.Field{
			zap.Any("assistants", assistants.Assistants),
		}

		log.Info("openai list assistants response", fields...)
	}
}
