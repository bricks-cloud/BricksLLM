package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
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
	EstimateChatCompletionPromptCostWithTokenCounts(r *goopenai.ChatCompletionRequest) (int, float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateChatCompletionStreamCostWithTokenCounts(model string, resp *goopenai.ChatCompletionStreamResponse) (int, float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
}

type validator interface {
	Validate(k *key.ResponseKey, promptCost float64) error
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

func getMiddleware(kms keyMemStorage, cpm CustomProvidersManager, prod, private bool, e estimator, v validator, ks keyStorage, log *zap.Logger, enc encrypter, rlm rateLimitManager, r recorder, prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] request is empty")
			c.Abort()
			return
		}

		if c.FullPath() == "/api/health" {
			c.Abort()
			return
		}

		cid := util.NewUuid()
		c.Set(correlationId, cid)
		start := time.Now()

		selectedProvider := "openai"

		customId := c.Request.Header.Get("X-CUSTOM-EVENT-ID")
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
				log.Info("response to openai proxy",
					zap.String(correlationId, c.GetString(correlationId)),
					zap.String("keyId", keyId),
					zap.Int("code", c.Writer.Status()),
					zap.String("method", c.Request.Method),
					zap.String("path", c.FullPath()),
					zap.Int("lantecyInMs", latency),
				)
			}

			stats.Incr("bricksllm.web.get_middleware.responses", []string{
				"status:" + strconv.Itoa(c.Writer.Status()),
			}, 1)

			evt := &event.Event{
				Id:                   util.NewUuid(),
				CreatedAt:            time.Now().Unix(),
				Tags:                 tags,
				KeyId:                keyId,
				CostInUsd:            c.GetFloat64("costInUsd"),
				Provider:             selectedProvider,
				Model:                c.GetString("model"),
				Status:               c.Writer.Status(),
				PromptTokenCount:     c.GetInt("promptTokenCount"),
				CompletionTokenCount: c.GetInt("completionTokenCount"),
				LatencyInMs:          latency,
				Path:                 c.FullPath(),
				Method:               c.Request.Method,
				CustomId:             customId,
			}

			err := r.RecordEvent(evt)
			if err != nil {
				stats.Incr("bricksllm.web.get_middleware.record_event_error", nil, 1)

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
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading request body", prod, cid, err)
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		var cost float64 = 0
		if strings.HasPrefix(c.FullPath(), "/api/custom/providers/:provider") {
			providerName := c.Param("provider")

			rc := cpm.GetRouteConfigFromMem(providerName, c.Param("wildcard"))
			cp := cpm.GetCustomProviderFromMem(providerName)
			if cp == nil {
				stats.Incr("bricksllm.web.get_middleware.provider_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] requested custom provider is not found")
				c.Abort()
				return
			}

			if rc == nil {
				stats.Incr("bricksllm.web.get_middleware.route_config_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] route config is not found")
				c.Abort()
				return
			}

			selectedProvider = cp.Provider

			c.Set("provider", cp)
			c.Set("route_config", rc)

			tks, err := countTokensFromJson(body, rc.RequestPromptLocation)
			if err != nil {
				logError(log, "error when counting tokens for custom provider request", prod, cid, err)
			}

			c.Set("promptTokenCount", tks)

			result := gjson.Get(string(body), rc.StreamLocation)

			fmt.Println(rc.StreamLocation)
			fmt.Println(result.IsBool())

			if result.IsBool() {
				fmt.Println(result.Bool())

				c.Set("streaming", result.Bool())
			}

			result = gjson.Get(string(body), rc.ModelLocation)
			if len(result.Str) != 0 {
				c.Set("model", result.Str)
			}
		}

		if c.FullPath() == "/api/providers/openai/v1/chat/completions" {
			ccr := &goopenai.ChatCompletionRequest{}
			err = json.Unmarshal(body, ccr)
			if err != nil {
				logError(log, "error when unmarshalling chat completion request", prod, cid, err)
				return
			}

			c.Set("model", ccr.Model)

			logRequest(log, prod, private, cid, ccr)

			tks, cost, err := e.EstimateChatCompletionPromptCostWithTokenCounts(ccr)
			if err != nil {
				stats.Incr("bricksllm.web.get_middleware.estimate_chat_completion_prompt_cost_with_token_counts_error", nil, 1)

				logError(log, "error when estimating prompt cost", prod, cid, err)
			}

			if ccr.Stream {
				c.Set("stream", true)
				c.Set("estimatedPromptCostInUsd", cost)
				c.Set("promptTokenCount", tks)
			}
		}

		if c.FullPath() == "/api/providers/openai/v1/embeddings" {
			er := &goopenai.EmbeddingRequest{}
			err = json.Unmarshal(body, er)
			if err != nil {
				logError(log, "error when unmarshalling embedding request", prod, cid, err)
				return
			}

			c.Set("model", er.Model.String())
			c.Set("encoding_format", string(er.EncodingFormat))

			logEmbeddingRequest(log, prod, private, cid, er)

			cost, err = e.EstimateEmbeddingsCost(er)
			if err != nil {
				stats.Incr("bricksllm.web.get_middleware.estimate_embeddings_cost_error", nil, 1)
				logError(log, "error when estimating embeddings cost", prod, cid, err)
			}
		}

		aid := c.Param("assistant_id")
		fid := c.Param("file_id")
		tid := c.Param("thread_id")
		mid := c.Param("message_id")
		rid := c.Param("run_id")
		sid := c.Param("step_id")
		md := c.Param("model")
		qm := map[string]string{}

		if val, ok := c.GetQuery("limit"); ok {
			qm["limit"] = val
		}

		if val, ok := c.GetQuery("order"); ok {
			qm["order"] = val
		}

		if val, ok := c.GetQuery("after"); ok {
			qm["after"] = val
		}

		if val, ok := c.GetQuery("before"); ok {
			qm["before"] = val
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodPost {
			logCreateAssistantRequest(log, body, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodGet {
			logRetrieveAssistantRequest(log, body, prod, cid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodPost {
			logModifyAssistantRequest(log, body, prod, private, cid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodDelete {
			logDeleteAssistantRequest(log, body, prod, cid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodGet {
			logListAssistantsRequest(log, prod, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodPost {
			logCreateAssistantFileRequest(log, body, prod, cid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodGet {
			logRetrieveAssistantFileRequest(log, prod, cid, fid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodDelete {
			logRetrieveAssistantFileRequest(log, prod, cid, fid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodGet {
			logListAssistantFilesRequest(log, prod, cid, aid, qm)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads" && c.Request.Method == http.MethodPost {
			logCreateThreadRequest(log, body, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodGet {
			logCreateThreadRequest(log, body, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodPost {
			logModifyThreadRequest(log, body, prod, cid, tid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodDelete {
			logDeleteThreadRequest(log, prod, cid, tid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodPost {
			logCreateMessageRequest(log, body, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodGet {
			logRetrieveMessageRequest(log, prod, cid, mid, tid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodPost {
			logModifyMessageRequest(log, body, prod, private, cid, tid, mid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodGet {
			logListMessagesRequest(log, body, prod, cid, aid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id" && c.Request.Method == http.MethodGet {
			logRetrieveMessageFileRequest(log, prod, cid, mid, tid, fid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files" && c.Request.Method == http.MethodGet {
			logListAssistantFilesRequest(log, prod, cid, aid, qm)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodPost {
			logCreateRunRequest(log, body, prod, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodGet {
			logRetrieveRunRequest(log, body, prod, cid, tid, rid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodPost {
			logModifyRunRequest(log, body, prod, cid, tid, rid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodGet {
			logListRunsRequest(log, body, prod, cid, tid, qm)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs" && c.Request.Method == http.MethodPost {
			logSubmitToolOutputsRequest(log, body, prod, cid, tid, rid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel" && c.Request.Method == http.MethodPost {
			logCancelARunRequest(log, body, prod, cid, tid, rid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/runs" && c.Request.Method == http.MethodPost {
			logCreateThreadAndRunRequest(log, body, prod, private, cid, tid, rid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id" && c.Request.Method == http.MethodGet {
			logRetrieveRunStepRequest(log, body, prod, cid, tid, rid, sid)
		}

		if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps" && c.Request.Method == http.MethodGet {
			logListRunStepsRequest(log, body, prod, cid, tid, rid, qm)
		}

		if c.FullPath() == "/api/providers/openai/v1/moderations" && c.Request.Method == http.MethodPost {
			logCreateModerationRequest(log, body, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/models" && c.Request.Method == http.MethodGet {
			logCreateModerationRequest(log, body, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodGet {
			logRetrieveModelRequest(log, body, prod, cid, md)
		}

		if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodDelete {
			logDeleteModelRequest(log, body, prod, cid, md)
		}

		err = v.Validate(kc, cost)
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

			logError(log, "error when validating api key", prod, cid, err)
			return
		}

		if len(kc.RateLimitUnit) != 0 {
			if err := rlm.Increment(kc.KeyId, kc.RateLimitUnit); err != nil {
				stats.Incr("bricksllm.web.get_middleware.rate_limit_increment_error", nil, 1)

				logError(log, "error when incrementing rate limit counter", prod, cid, err)
			}
		}

		c.Next()
	}
}
