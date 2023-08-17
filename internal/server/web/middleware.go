package web

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/gin-gonic/gin"
)

type KeyMemStorage interface {
	GetKey(hash string) *key.ResponseKey
}

type KeyStorage interface {
	UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error)
}

type Estimator interface {
	EstimateChatCompletionPromptCost(r *openai.ChatCompletionRequest) (float64, error)
}

type Validator interface {
	Validate(k *key.ResponseKey, promptCost float64, model string) error
}

const apiHeader string = "X-Api-Key"

func KeyValidator(kms KeyMemStorage, e Estimator, v Validator, ks KeyStorage, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		apiKey := c.Request.Header.Get(apiKeyHeader)
		if len(apiKey) == 0 {
			c.Status(http.StatusUnauthorized)
			return
		}

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

			if _, ok := err.(ExpirationError); ok {
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

			if _, ok := err.(RateLimitError); ok {
				c.JSON(http.StatusTooManyRequests, err.Error())
				return
			}

			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	}
}
