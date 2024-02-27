package proxy

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
	"github.com/bricks-cloud/bricksllm/internal/message"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	goopenai "github.com/sashabaranov/go-openai"
)

type keyMemStorage interface {
	GetKey(hash string) *key.ResponseKey
}

type keyStorage interface {
	UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error)
}

type estimator interface {
	EstimateTranscriptionCost(secs float64, model string) (float64, error)
	EstimateSpeechCost(input string, model string) (float64, error)
	EstimateChatCompletionPromptCostWithTokenCounts(r *goopenai.ChatCompletionRequest) (int, float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateChatCompletionStreamCostWithTokenCounts(model, content string) (int, float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
	EstimateChatCompletionPromptTokenCounts(model string, r *goopenai.ChatCompletionRequest) (int, error)
}

type azureEstimator interface {
	EstimateChatCompletionStreamCostWithTokenCounts(model, content string) (int, float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimatePromptCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
}

type authenticator interface {
	AuthenticateHttpRequest(req *http.Request) (*key.ResponseKey, []*provider.Setting, error)
}

type validator interface {
	Validate(k *key.ResponseKey, promptCost float64) error
}

type rateLimitManager interface {
	Increment(keyId string, timeUnit key.TimeUnit) error
}

type accessCache interface {
	GetAccessStatus(key string) bool
}

func JSON(c *gin.Context, code int, message string) {
	c.JSON(code, &goopenai.ErrorResponse{
		Error: &goopenai.APIError{
			Message: message,
			Code:    strconv.Itoa(code),
		},
	})
}

type notAuthorizedError interface {
	Authenticated()
}

type notFoundError interface {
	NotFound()
}

type publisher interface {
	Publish(message.Message)
}

func getProvider(c *gin.Context) string {
	existing := c.GetString("provider")
	if len(existing) != 0 {
		return existing
	}

	parts := strings.Split(c.FullPath(), "/")

	spaceRemoved := []string{}

	for _, part := range parts {
		if len(part) != 0 {
			spaceRemoved = append(spaceRemoved, part)
		}
	}

	if strings.HasPrefix(c.FullPath(), "/api/providers/") {
		if len(spaceRemoved) >= 3 {
			return spaceRemoved[2]
		}
	}

	if strings.HasPrefix(c.FullPath(), "/api/custom/providers/") {
		return c.Param("provider")
	}

	return ""
}

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func getMiddleware(kms keyMemStorage, cpm CustomProvidersManager, rm routeManager, a authenticator, prod, private bool, e estimator, ae anthropicEstimator, aoe azureEstimator, v validator, ks keyStorage, log *zap.Logger, rlm rateLimitManager, pub publisher, prefix string, ac accessCache) gin.HandlerFunc {
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

		blw := &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		cid := util.NewUuid()
		c.Set(correlationId, cid)
		start := time.Now()

		enrichedEvent := &event.EventWithRequestAndContent{}
		requestBytes := []byte(`{}`)
		responseBytes := []byte(`{}`)
		userId := ""

		customId := c.Request.Header.Get("X-CUSTOM-EVENT-ID")
		defer func() {
			dur := time.Since(start)
			latency := int(dur.Milliseconds())

			if !prod {
				log.Sugar().Infof("%s | %d | %s | %s | %dms", prefix, c.Writer.Status(), c.Request.Method, c.FullPath(), latency)
			}

			keyId := ""
			tags := []string{}

			if enrichedEvent.Key != nil {
				keyId = enrichedEvent.Key.KeyId
				tags = enrichedEvent.Key.Tags
			}

			stats.Timing("bricksllm.proxy.get_middleware.proxy_latency_in_ms", dur, nil, 1)

			selectedProvider := getProvider(c)

			if prod {
				log.Info("response to proxy",
					zap.String(correlationId, c.GetString(correlationId)),
					zap.String("provider", selectedProvider),
					zap.String("keyId", keyId),
					zap.Int("code", c.Writer.Status()),
					zap.String("method", c.Request.Method),
					zap.String("path", c.FullPath()),
					zap.Int("lantecyInMs", latency),
				)
			}

			stats.Incr("bricksllm.proxy.get_middleware.responses", []string{
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
				Request:              requestBytes,
				Response:             responseBytes,
				UserId:               userId,
			}

			enrichedEvent.Event = evt
			content := c.GetString("content")
			if len(content) != 0 {
				enrichedEvent.Content = content
			}

			resp, ok := c.Get("response")
			if ok {
				enrichedEvent.Response = resp
			}

			pub.Publish(message.Message{
				Type: "event",
				Data: enrichedEvent,
			})
		}()

		if len(c.FullPath()) == 0 {
			stats.Incr("bricksllm.proxy.get_middleware.route_does_not_exist", nil, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] route not supported")
			c.Abort()
			return
		}

		kc, settings, err := a.AuthenticateHttpRequest(c.Request)
		enrichedEvent.Key = kc
		_, ok := err.(notAuthorizedError)
		if ok {
			stats.Incr("bricksllm.proxy.get_middleware.authentication_error", nil, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] not authorized")
			c.Abort()
			return
		}

		_, ok = err.(notFoundError)
		if ok {
			stats.Incr("bricksllm.proxy.get_middleware.not_found_error", nil, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] route not found")
			c.Abort()
			return
		}

		if err != nil {
			stats.Incr("bricksllm.proxy.get_middleware.authenticate_http_request_error", nil, 1)
			logError(log, "error when authenticating http requests", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] internal authentication error")
			c.Abort()
			return
		}

		c.Set("key", kc)
		c.Set("settings", settings)

		if len(settings) >= 1 {
			if strings.HasPrefix(c.FullPath(), "/api/providers/azure/openai") {
				selected := settings[0]

				if selected != nil && len(selected.Setting["resourceName"]) != 0 {
					c.Set("resourceName", selected.Setting["resourceName"])
				}
			}
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading request body", prod, cid, err)
			return
		}

		if kc.ShouldLogRequest {
			requestBytes = body
		}

		if c.Request.Method != http.MethodGet {
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		// var cost float64 = 0

		if c.FullPath() == "/api/providers/anthropic/v1/complete" {
			logCompletionRequest(log, body, prod, private, cid)

			cr := &anthropic.CompletionRequest{}
			err = json.Unmarshal(body, cr)
			if err != nil {
				logError(log, "error when unmarshalling anthropic completion request", prod, cid, err)
				return
			}

			if cr.Metadata != nil {
				userId = cr.Metadata.UserId
			}

			enrichedEvent.Request = cr

			// tks := ae.Count(cr.Prompt)
			// tks += anthropicPromptMagicNum
			// c.Set("promptTokenCount", tks)

			// model := cr.Model
			// cost, err = ae.EstimatePromptCost(model, tks)
			// if err != nil {
			// 	logError(log, "error when estimating anthropic completion prompt cost", prod, cid, err)
			// }

			if cr.Stream {
				c.Set("stream", cr.Stream)
				// c.Set("estimatedPromptCostInUsd", cost)
			}

			if len(cr.Model) != 0 {
				c.Set("model", cr.Model)
			}
		}

		if strings.HasPrefix(c.FullPath(), "/api/custom/providers/:provider") {
			providerName := c.Param("provider")

			rc := cpm.GetRouteConfigFromMem(providerName, c.Param("wildcard"))
			enrichedEvent.RouteConfig = rc

			cp := cpm.GetCustomProviderFromMem(providerName)
			if cp == nil {
				stats.Incr("bricksllm.proxy.get_middleware.provider_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] requested custom provider is not found")
				c.Abort()
				return
			}

			if rc == nil {
				stats.Incr("bricksllm.proxy.get_middleware.route_config_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] route config is not found")
				c.Abort()
				return
			}

			c.Set("provider", cp)
			c.Set("route_config", rc)

			enrichedEvent.Request = body

			customResponse, ok := c.Get("response")
			if ok {
				enrichedEvent.Response = customResponse
			}

			// tks, err := countTokensFromJson(body, rc.RequestPromptLocation)
			// if err != nil {
			// 	logError(log, "error when counting tokens for custom provider request", prod, cid, err)
			// }

			// c.Set("promptTokenCount", tks)

			result := gjson.Get(string(body), rc.StreamLocation)

			if result.IsBool() {
				c.Set("stream", result.Bool())
			}

			result = gjson.Get(string(body), rc.ModelLocation)
			if len(result.Str) != 0 {
				c.Set("model", result.Str)
			}
		}

		if strings.HasPrefix(c.FullPath(), "/api/routes") {
			r := c.Param("route")
			rc := rm.GetRouteFromMemDb(r)

			if rc == nil {
				stats.Incr("bricksllm.proxy.get_middleware.route_config_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] route config is not found")
				c.Abort()
				return
			}

			c.Set("route_config", rc)

			if rc.ShouldRunEmbeddings() {
				er := &goopenai.EmbeddingRequest{}
				err = json.Unmarshal(body, er)
				if err != nil {
					logError(log, "error when unmarshalling route embedding request", prod, cid, err)
					return
				}

				userId = er.User

				if rc.CacheConfig != nil && rc.CacheConfig.Enabled {
					c.Set("cache_key", route.ComputeCacheKeyForEmbeddingsRequest(r, er))
				}

				c.Set("encoding_format", string(er.EncodingFormat))

				logEmbeddingRequest(log, prod, private, cid, er)
			}

			if !rc.ShouldRunEmbeddings() {
				ccr := &goopenai.ChatCompletionRequest{}

				err = json.Unmarshal(body, ccr)
				if err != nil {
					logError(log, "error when unmarshalling route chat completion request", prod, cid, err)
					return
				}

				userId = ccr.User
				enrichedEvent.Request = ccr

				logRequest(log, prod, private, cid, ccr)

				if ccr.Stream {
					stats.Incr("bricksllm.proxy.get_middleware.streaming_not_allowed", nil, 1)
					JSON(c, http.StatusForbidden, "[BricksLLM] streaming is not allowed")
					c.Abort()
					return
				}

				if rc.CacheConfig != nil && rc.CacheConfig.Enabled {
					c.Set("cache_key", route.ComputeCacheKeyForChatCompletionRequest(r, ccr))
				}
			}
		}

		if c.FullPath() == "/api/providers/azure/openai/deployments/:deployment_id/chat/completions" {
			ccr := &goopenai.ChatCompletionRequest{}
			err = json.Unmarshal(body, ccr)
			if err != nil {
				logError(log, "error when unmarshalling azure openai chat completion request", prod, cid, err)
				return
			}

			userId = ccr.User

			enrichedEvent.Request = ccr

			logRequest(log, prod, private, cid, ccr)

			// tks, err := e.EstimateChatCompletionPromptTokenCounts("gpt-3.5-turbo", ccr)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_middleware.estimate_chat_completion_prompt_token_counts_error", nil, 1)
			// 	logError(log, "error when estimating prompt cost", prod, cid, err)
			// }

			if ccr.Stream {
				c.Set("stream", true)
				// c.Set("promptTokenCount", tks)
			}
		}

		if c.FullPath() == "/api/providers/azure/openai/deployments/:deployment_id/embeddings" {
			er := &goopenai.EmbeddingRequest{}
			err = json.Unmarshal(body, er)
			if err != nil {
				logError(log, "error when unmarshalling azure openai embedding request", prod, cid, err)
				return
			}

			userId = er.User

			c.Set("model", "ada")
			c.Set("encoding_format", string(er.EncodingFormat))

			logEmbeddingRequest(log, prod, private, cid, er)

			// cost, err = aoe.EstimateEmbeddingsCost(er)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_middleware.estimate_azure_openai_embeddings_cost_error", nil, 1)
			// 	logError(log, "error when estimating azure openai embeddings cost", prod, cid, err)
			// }
		}

		if c.FullPath() == "/api/providers/openai/v1/chat/completions" {
			ccr := &goopenai.ChatCompletionRequest{}
			err = json.Unmarshal(body, ccr)
			if err != nil {
				logError(log, "error when unmarshalling chat completion request", prod, cid, err)
				return
			}

			userId = ccr.User

			enrichedEvent.Request = ccr

			c.Set("model", ccr.Model)

			logRequest(log, prod, private, cid, ccr)

			// tks, cost, err := e.EstimateChatCompletionPromptCostWithTokenCounts(ccr)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_middleware.estimate_chat_completion_prompt_cost_with_token_counts_error", nil, 1)

			// 	logError(log, "error when estimating prompt cost", prod, cid, err)
			// }

			if ccr.Stream {
				c.Set("stream", true)
				// c.Set("estimatedPromptCostInUsd", cost)
				// c.Set("promptTokenCount", tks)
			}
		}

		if c.FullPath() == "/api/providers/openai/v1/embeddings" {
			er := &goopenai.EmbeddingRequest{}
			err = json.Unmarshal(body, er)
			if err != nil {
				logError(log, "error when unmarshalling embedding request", prod, cid, err)
				return
			}

			userId = er.User

			c.Set("model", string(er.Model))
			c.Set("encoding_format", string(er.EncodingFormat))

			logEmbeddingRequest(log, prod, private, cid, er)

			// cost, err = e.EstimateEmbeddingsCost(er)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_middleware.estimate_embeddings_cost_error", nil, 1)
			// 	logError(log, "error when estimating embeddings cost", prod, cid, err)
			// }
		}

		if c.FullPath() == "/api/providers/openai/v1/images/generations" && c.Request.Method == http.MethodPost {
			ir := &goopenai.ImageRequest{}
			err := json.Unmarshal(body, ir)
			if err != nil {
				logError(log, "error when unmarshalling create image request", prod, cid, err)
				return
			}

			c.Set("model", ir.Model)

			if len(ir.Model) == 0 {
				c.Set("model", "dall-e-2")
			}

			c.Set("model", ir.Model)
			logCreateImageRequest(log, ir, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/images/edits" && c.Request.Method == http.MethodPost {
			prompt := c.PostForm("model")
			model := c.PostForm("model")
			size := c.PostForm("size")
			user := c.PostForm("user")

			userId = user

			responseFormat := c.PostForm("response_format")
			n, _ := strconv.Atoi(c.PostForm("n"))

			c.Set("model", model)

			if len(model) == 0 {
				c.Set("model", "dall-e-2")
			}

			logEditImageRequest(log, prompt, model, n, size, responseFormat, user, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/images/variations" && c.Request.Method == http.MethodPost {
			model := c.PostForm("model")
			size := c.PostForm("size")
			user := c.PostForm("user")

			userId = user

			responseFormat := c.PostForm("response_format")
			n, _ := strconv.Atoi(c.PostForm("n"))

			c.Set("model", model)

			if len(model) == 0 {
				c.Set("model", "dall-e-2")
			}

			logImageVariationsRequest(log, model, n, size, responseFormat, user, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/audio/speech" && c.Request.Method == http.MethodPost {
			sr := &goopenai.CreateSpeechRequest{}
			err := json.Unmarshal(body, sr)
			if err != nil {
				logError(log, "error when unmarshalling create speech request", prod, cid, err)
				return
			}

			enrichedEvent.Request = sr

			c.Set("model", string(sr.Model))

			logCreateSpeechRequest(log, sr, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/audio/transcriptions" && c.Request.Method == http.MethodPost {
			model := c.PostForm("model")
			language := c.PostForm("language")
			prompt := c.PostForm("prompt")
			responseFormat := c.PostForm("response_format")
			temperature := c.PostForm("temperature")

			c.Set("model", model)

			converted, _ := strconv.ParseFloat(temperature, 64)
			logCreateTranscriptionRequest(log, model, language, prompt, responseFormat, converted, prod, private, cid)
		}

		if c.FullPath() == "/api/providers/openai/v1/audio/translations" && c.Request.Method == http.MethodPost {
			model := c.PostForm("model")
			prompt := c.PostForm("prompt")
			responseFormat := c.PostForm("response_format")
			temperature := c.PostForm("temperature")

			c.Set("model", model)

			converted, _ := strconv.ParseFloat(temperature, 64)
			logCreateTranslationRequest(log, model, prompt, responseFormat, converted, prod, private, cid)
		}

		if len(kc.AllowedPaths) != 0 && !containsPath(kc.AllowedPaths, c.FullPath(), c.Request.Method) {
			stats.Incr("bricksllm.proxy.get_middleware.path_not_allowed", nil, 1)
			JSON(c, http.StatusForbidden, "[BricksLLM] path is not allowed")
			c.Abort()
			return
		}

		model := c.GetString("model")
		if !isModelAllowed(model, settings) {
			stats.Incr("bricksllm.proxy.get_middleware.model_not_allowed", nil, 1)
			JSON(c, http.StatusForbidden, "[BricksLLM] model is not allowed")
			c.Abort()
			return
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

		if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodGet {
			logListFilesRequest(log, prod, cid, qm)
		}

		if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodPost {
			purpose := c.PostForm("purpose")
			logUploadFileRequest(log, prod, cid, purpose)
		}

		if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodDelete {
			logDeleteFileRequest(log, prod, cid, fid)
		}

		if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodGet {
			logRetrieveFileRequest(log, prod, cid, fid)
		}

		if c.FullPath() == "/api/providers/openai/v1/files/:file_id/content" && c.Request.Method == http.MethodGet {
			logRetrieveFileContentRequest(log, body, prod, cid, fid)
		}

		if ac.GetAccessStatus(kc.KeyId) {
			stats.Incr("bricksllm.proxy.get_middleware.rate_limited", nil, 1)
			JSON(c, http.StatusTooManyRequests, "[BricksLLM] too many requests")
			c.Abort()
			return
		}

		c.Next()

		if kc.ShouldLogResponse {
			responseBytes = blw.body.Bytes()
		}
	}
}

func contains(arr []string, target string) bool {
	for _, str := range arr {
		if str == target {
			return true
		}
	}

	return false
}

func isModelAllowed(model string, settings []*provider.Setting) bool {
	if len(model) == 0 {
		return true
	}

	for _, setting := range settings {
		if len(setting.AllowedModels) == 0 {
			return true
		}

		if contains(setting.AllowedModels, model) {
			return true
		}
	}

	return false
}

func containsPath(arr []key.PathConfig, path, method string) bool {
	for _, pc := range arr {
		if pc.Path == path && pc.Method == method {
			return true
		}
	}

	return false
}
