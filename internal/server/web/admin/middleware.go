package admin

import (
	"time"

	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func getAdminLoggerMiddleware(log *zap.Logger, prefix string, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(correlationId, util.NewUuid())
		start := time.Now()
		c.Next()
		latency := time.Now().Sub(start).Milliseconds()
		if !prod {
			log.Sugar().Infof("%s | %d | %s | %s | %dms", prefix, c.Writer.Status(), c.Request.Method, c.FullPath(), latency)
		}

		if prod {
			log.Info("request to admin management api",
				zap.String(correlationId, c.GetString(correlationId)),
				zap.Int("code", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", c.FullPath()),
				zap.Int64("lantecyInMs", latency),
			)
		}
	}
}
