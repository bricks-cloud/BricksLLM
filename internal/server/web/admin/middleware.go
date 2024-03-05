package admin

import (
	"time"

	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func getAdminLoggerMiddleware(log *zap.Logger, prefix string, prod bool, adminPass string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(adminPass) != 0 && c.Request.Header.Get("X-API-KEY") != adminPass {
			c.Status(200)
			c.Abort()
			return
		}

		c.Set(correlationId, util.NewUuid())
		start := time.Now()
		c.Next()
		latency := time.Since(start).Milliseconds()
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
