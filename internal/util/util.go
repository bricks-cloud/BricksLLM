package util

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	log = "log"
)

func NewUuid() string {
	return uuid.New().String()
}

func GetLogFromCtx(c *gin.Context) *zap.Logger {
	logWithCid := c.Request.Context().Value(log).(*zap.Logger)
	return logWithCid
}
