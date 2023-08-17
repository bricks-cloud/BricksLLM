package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/gin-gonic/gin"
)

type rateLimitError interface {
	Error() string
	RateLimit()
}

type expirationError interface {
	Error() string
	Reason() string
}

type keyMemStorage interface {
	GetKey(hash string) *key.ResponseKey
}

type keyStorage interface {
	UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error)
}

type estimator interface {
	EstimateChatCompletionPromptCost(r *openai.ChatCompletionRequest) (float64, error)
}

type validator interface {
	Validate(k *key.ResponseKey, promptCost float64, model string) error
}

const apiHeader string = "X-Api-Key"

func getKeyValidator(kms keyMemStorage, e estimator, v validator, ks keyStorage, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		split := strings.Split(c.Request.Header.Get("Authorization"), "Bearer ")
		if len(split) < 2 || len(split[1]) == 0 {
			c.Status(http.StatusUnauthorized)
			return
		}

		apiKey := split[1]
		kc := kms.GetKey(apiKey)
		if kc == nil {
			c.Status(http.StatusUnauthorized)
			return
		}

		bytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		ccr := &openai.ChatCompletionRequest{}
		err = json.Unmarshal(bytes, ccr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		cost, err := e.EstimateChatCompletionPromptCost(ccr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		err = v.Validate(kc, cost, ccr.Model)
		if err != nil {
			if _, ok := err.(ValidationError); ok {
				c.JSON(http.StatusUnauthorized, err.Error())
				return
			}

			if _, ok := err.(expirationError); ok {
				truePtr := true
				_, err = ks.UpdateKey(kc.KeyId, &key.UpdateKey{
					Revoked: &truePtr,
				})

				if err != nil {
					log.Debugf("error when updating revoking the api key %s: %v", kc.KeyId, err)
				}

				c.JSON(http.StatusUnauthorized, err.Error())
				return
			}

			if _, ok := err.(rateLimitError); ok {
				c.JSON(http.StatusTooManyRequests, err.Error())
				return
			}

			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	}
}
