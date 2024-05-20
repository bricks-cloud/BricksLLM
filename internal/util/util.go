package util

import (
	"context"

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
