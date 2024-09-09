package util

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ctxKey string

const (
	STRING_CORRELATION_ID string = "correlationId"
	STRING_LOG            ctxKey = "log"
)

func NewUuid() string {
	return uuid.New().String()
}

func SetLogToCtx(c *gin.Context, logger *zap.Logger) {
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), STRING_LOG, logger))
}

func GetLogFromCtx(c *gin.Context) *zap.Logger {
	logWithCid := c.Request.Context().Value(STRING_LOG).(*zap.Logger)
	return logWithCid
}

func ConvertAnyToStr(input any) (string, error) {
	converted := ""

	if str, ok := input.(string); ok {
		converted += str
	} else if arr, ok := input.([]interface{}); ok {
		for _, unknown := range arr {
			str, ok := unknown.(string)
			if ok {
				converted += str
				continue
			}

			return "", errors.New("input array contains a non string entry")
		}
	} else {
		return "", errors.New("input is neither string nor an array of strings")
	}

	return converted, nil
}

func TranslateBedrockModelToAnthropicModel(model string) string {
	if strings.HasPrefix(model, "anthropic.claude-v2") {
		return "claude"
	}

	if strings.HasPrefix(model, "anthropic.claude-3-haiku") {
		return "claude-3-haiku"
	}

	if strings.HasPrefix(model, "anthropic.claude-3-sonnet") {
		return "claude-3-sonnet"
	}

	if strings.HasPrefix(model, "anthropic.claude-3-opus") {
		return "claude-3-opus"
	}

	if strings.HasPrefix(model, "anthropic.claude-3-5-sonnet") {
		return "claude-3.5-sonnet"
	}

	if strings.HasPrefix(model, "anthropic.claude-instant") {
		return "claude-instant"
	}

	return model
}
