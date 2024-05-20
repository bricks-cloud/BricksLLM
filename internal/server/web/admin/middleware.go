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

		cid := util.NewUuid()
		c.Set(util.STRING_CORRELATION_ID, cid)
		logWithCid := log.With(zap.String(util.STRING_CORRELATION_ID, cid))
		util.SetLogToCtx(c, logWithCid)

		start := time.Now()
		c.Next()
		latency := time.Since(start).Milliseconds()
		if !prod {
			logWithCid.Sugar().Infof("%s | %d | %s | %s | %dms", prefix, c.Writer.Status(), c.Request.Method, c.FullPath(), latency)
		}

		if prod {
			logWithCid.Info("request to admin management api",
				zap.Int("code", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", c.FullPath()),
				zap.Int64("lantecyInMs", latency),
			)
		}
	}
}
