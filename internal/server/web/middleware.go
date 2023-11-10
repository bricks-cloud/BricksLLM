package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	goopenai "github.com/sashabaranov/go-openai"
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
	EstimateChatCompletionPromptCost(r *goopenai.ChatCompletionRequest) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
}

type validator interface {
	Validate(k *key.ResponseKey, promptCost float64, model string) error
}

type rateLimitManager interface {
	Increment(keyId string, timeUnit key.TimeUnit) error
}

type encrypter interface {
	Encrypt(secret string) string
}

func JSON(c *gin.Context, code int, message string) {
	c.JSON(code, &goopenai.ErrorResponse{
		Error: &goopenai.APIError{
			Message: message,
			Code:    code,
		},
	})
}

func getMiddleware(kms keyMemStorage, prod, private bool, e estimator, v validator, ks keyStorage, log *zap.Logger, enc encrypter, rlm rateLimitManager, r recorder, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] request is empty")
			c.Abort()
			return
		}

		cid := util.NewUuid()
		c.Set(correlationId, cid)
		start := time.Now()

		defer func() {
			dur := time.Now().Sub(start)
			latency := int(dur.Milliseconds())
			raw, exists := c.Get("key")
			var kc *key.ResponseKey
			if exists {
				kc = raw.(*key.ResponseKey)
			}

			if !prod {
				log.Sugar().Infof("%s | %d | %s | %s | %dms", prefix, c.Writer.Status(), c.Request.Method, c.FullPath(), latency)
			}

			keyId := ""
			tags := []string{}

			if kc != nil {
				keyId = kc.KeyId
				tags = kc.Tags
			}

			stats.Timing("bricksllm.web.get_middleware.proxy_latency_in_ms", dur, nil, 1)

			if prod {
				log.Info("request to openai proxy",
					zap.String(correlationId, c.GetString(correlationId)),
					zap.String("keyId", keyId),
					zap.Int("code", c.Writer.Status()),
					zap.String("method", c.Request.Method),
					zap.String("path", c.FullPath()),
					zap.Int("lantecyInMs", latency),
				)
			}

			stats.Incr("bricksllm.web.get_middleware.responses", []string{
				strconv.Itoa(c.Writer.Status()),
			}, 1)

			evt := &event.Event{
				Id:                   util.NewUuid(),
				CreatedAt:            time.Now().Unix(),
				Tags:                 tags,
				KeyId:                keyId,
				CostInUsd:            c.GetFloat64("costInUsd"),
				Provider:             provider.OpenAiProvider,
				Model:                c.GetString("model"),
				Status:               c.Writer.Status(),
				PromptTokenCount:     c.GetInt("promptTokenCount"),
				CompletionTokenCount: c.GetInt("completionTokenCount"),
				LatencyInMs:          latency,
			}

			err := r.RecordEvent(evt)
			if err != nil {
				stats.Incr("bricksllm.web.get_middleware.record_event_error", []string{
					strconv.Itoa(c.Writer.Status()),
				}, 1)

				logError(log, "error when recording openai event", prod, cid, err)
			}
		}()

		split := strings.Split(c.Request.Header.Get("Authorization"), "Bearer ")
		if len(split) < 2 || len(split[1]) == 0 {
			stats.Incr("bricksllm.web.get_middleware.missing_bearer_token", nil, 1)

			JSON(c, http.StatusUnauthorized, "[BricksLLM] bearer token is not present")
			c.Abort()
			return
		}

		apiKey := split[1]
		hash := enc.Encrypt(apiKey)

		kc := kms.GetKey(hash)
		if kc == nil {
			stats.Incr("bricksllm.web.get_middleware.api_key_is_not_authorized", nil, 1)

			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is unauthorized")
			c.Abort()
			return
		}

		if kc.Revoked {
			stats.Incr("bricksllm.web.get_middleware.api_key_revoked", nil, 1)

			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key has been revoked")
			c.Abort()
			return
		}

		c.Set("key", kc)
		id := c.GetString(correlationId)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading request body", prod, id, err)
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		ccr := &goopenai.ChatCompletionRequest{}
		err = json.Unmarshal(body, ccr)
		if err != nil {
			logError(log, "error when unmarshalling json", prod, id, err)
			return
		}

		c.Set("model", ccr.Model)

		logRequest(log, prod, private, id, ccr)

		cost, err := e.EstimateChatCompletionPromptCost(ccr)
		if err != nil {
			stats.Incr("bricksllm.web.get_middleware.estimate_chat_completion_prompt_cost_error", nil, 1)

			logError(log, "error when estimating prompt cost", prod, id, err)
		}

		err = v.Validate(kc, cost, ccr.Model)
		if err != nil {
			stats.Incr("bricksllm.web.get_middleware.validation_error", nil, 1)

			if _, ok := err.(expirationError); ok {
				stats.Incr("bricksllm.web.get_middleware.key_expired", nil, 1)

				truePtr := true
				_, err = ks.UpdateKey(kc.KeyId, &key.UpdateKey{
					Revoked:       &truePtr,
					RevokedReason: "Key has expired or exceeded set spend limit",
				})

				if err != nil {
					stats.Incr("bricksllm.web.get_middleware.update_key_error", nil, 1)
					log.Sugar().Debugf("error when updating revoking the api key %s: %v", kc.KeyId, err)
				}

				JSON(c, http.StatusUnauthorized, "[BricksLLM] key has expired")
				c.Abort()
				return
			}

			if _, ok := err.(rateLimitError); ok {
				stats.Incr("bricksllm.web.get_middleware.rate_limited", nil, 1)
				JSON(c, http.StatusTooManyRequests, "[BricksLLM] too many requests")
				c.Abort()
				return
			}

			logError(log, "error when validating api key", prod, id, err)
			return
		}

		if len(kc.RateLimitUnit) != 0 {
			if err := rlm.Increment(kc.KeyId, kc.RateLimitUnit); err != nil {
				stats.Incr("bricksllm.web.get_middleware.rate_limit_increment_error", nil, 1)

				logError(log, "error when incrementing rate limit counter", prod, id, err)
			}
		}

		c.Next()
	}
}
