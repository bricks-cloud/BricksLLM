package proxy

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func getTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] request is empty")
			c.Abort()
			return
		}

		timeoutHeader := c.GetHeader("x-request-timeout")
		parsedTimeout := timeout
		if len(timeoutHeader) != 0 {
			parsed, err := time.ParseDuration(timeoutHeader)
			if err != nil {
				JSON(c, http.StatusBadRequest, "[BricksLLM] invalid timeout")
				c.Abort()
				return
			}

			parsedTimeout = parsed
		}

		c.Set("requestTimeout", parsedTimeout)
	}
}
