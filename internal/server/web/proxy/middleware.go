package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"strconv"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/provider/azure"
	"github.com/bricks-cloud/bricksllm/internal/provider/deepinfra"
	"github.com/bricks-cloud/bricksllm/internal/provider/xcustom"
	gopointer "github.com/sergei-bronnikov/go-pointer"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/message"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/bricks-cloud/bricksllm/internal/provider/vllm"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/user"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"

	responsesOpenai "github.com/openai/openai-go/v3/responses"
	goopenai "github.com/sashabaranov/go-openai"
)

type keyMemStorage interface {
	GetKey(hash string) *key.ResponseKey
}

type userManager interface {
	GetUsers(tags, keyIds, userIds []string, offset int, limit int) ([]*user.User, error)
}

type keyStorage interface {
	UpdateKey(id string, uk *key.UpdateKey) (*key.ResponseKey, error)
}

type estimator interface {
	EstimateTranscriptionCost(secs float64, model string, usage *openai.TranscriptionResponseUsage) (float64, error)
	EstimateSpeechCost(input string, model string) (float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateChatCompletionStreamCostWithTokenCounts(model string, promptTks int, content string) (int, float64, float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
	EstimateChatCompletionPromptTokenCounts(model string, r *goopenai.ChatCompletionRequest) (int, error)
	EstimateResponseApiTotalCost(model string, usage responsesOpenai.ResponseUsage) (float64, error)
	EstimateResponseApiToolCallsCost(tools []responsesOpenai.ToolUnion, model string) (float64, error)
	EstimateResponseApiToolCreateContainerCost(req *openai.ResponseRequest) (float64, error)
	EstimateVideoCost(metadata *openai.VideoResponseMetadata) (float64, error)
}

type azureEstimator interface {
	EstimateChatCompletionStreamCostWithTokenCounts(model, content string) (int, float64, error)
	EstimateEmbeddingsCost(r *goopenai.EmbeddingRequest) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimatePromptCost(model string, tks int) (float64, error)
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
}

type deepinfraEstimator interface {
	EstimateEmbeddingsInputCost(model string, tks int) (float64, error)
}

type authenticator interface {
	AuthenticateHttpRequest(req *http.Request, xCustomProviderId string) (*key.ResponseKey, []*provider.Setting, error)
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

type userAccessCache interface {
	GetAccessStatus(userId string) bool
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

type blockedError interface {
	Error() string
	Blocked()
}

type warnedError interface {
	Error() string
	Warnings()
}

type redactedError interface {
	Error() string
	Redacted()
}

type publisher interface {
	Publish(message.Message)
}

func getProvider(c *gin.Context, fullPath string) string {
	existing := c.GetString("provider")
	if len(existing) != 0 {
		return existing
	}

	parts := strings.Split(fullPath, "/")

	spaceRemoved := []string{}

	for _, part := range parts {
		if len(part) != 0 {
			spaceRemoved = append(spaceRemoved, part)
		}
	}

	if strings.HasPrefix(fullPath, "/api/providers/") {
		if len(spaceRemoved) >= 3 {
			return spaceRemoved[2]
		}
	}

	if strings.HasPrefix(fullPath, "/api/custom/providers/") {
		return c.Param("provider")
	}

	if strings.HasPrefix(fullPath, "/api/routes/") {
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

type CustomPolicyDetector interface {
	Detect(input []string, requirements []string) (bool, error)
}

func getMiddleware(cpm CustomProvidersManager, rm routeManager, pm PoliciesManager, a authenticator, prod, private bool, log *zap.Logger, pub publisher, prefix string, ac accessCache, uac userAccessCache, client http.Client, scanner Scanner, cd CustomPolicyDetector, um userManager, removeUserAgent bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] request is empty")
			c.Abort()
			return
		}

		fullPath := c.FullPath()
		method := c.Request.Method

		if fullPath == "/api/health" {
			c.Abort()
			return
		}

		if removeUserAgent {
			c.Set("removeUserAgent", removeUserAgent)
		}

		var blw *responseWriter

		cid := util.NewUuid()
		c.Set(util.STRING_CORRELATION_ID, cid)
		logWithCid := log.With(zap.String(util.STRING_CORRELATION_ID, cid))
		util.SetLogToCtx(c, logWithCid)

		start := time.Now()
		c.Set("startTime", start)

		enrichedEvent := &event.EventWithRequestAndContent{}
		requestBytes := []byte(`{}`)
		responseBytes := []byte(`{}`)
		userId := ""

		var policyInput any = nil

		customId := c.Request.Header.Get("X-CUSTOM-EVENT-ID")

		metadataBytes := []byte(`{}`)
		metadata := c.Request.Header.Get("X-METADATA")

		defer func() {
			dur := time.Since(start)
			latency := int(dur.Milliseconds())

			if !prod {
				logWithCid.Sugar().Infof("%s | %d | %s | %s | %dms", prefix, c.Writer.Status(), method, fullPath, latency)
			}

			keyId := ""
			tags := []string{}

			if enrichedEvent.Key != nil {
				keyId = enrichedEvent.Key.KeyId
				tags = enrichedEvent.Key.Tags
			}

			if len(metadata) != 0 {
				metadataBytes = []byte(metadata)
			}

			telemetry.Timing("bricksllm.proxy.get_middleware.proxy_latency_in_ms", dur, nil, 1)

			selectedProvider := getProvider(c, fullPath)

			if prod {
				logWithCid.Info("response to proxy",
					zap.String("provider", selectedProvider),
					zap.String("keyId", keyId),
					zap.Int("code", c.Writer.Status()),
					zap.String("method", method),
					zap.String("path", fullPath),
					zap.Int("lantecyInMs", latency),
				)
			}

			telemetry.Incr("bricksllm.proxy.get_middleware.responses", []string{
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
				Path:                 c.Request.URL.Path,
				Method:               method,
				CustomId:             customId,
				Request:              requestBytes,
				Response:             responseBytes,
				UserId:               userId,
				PolicyId:             c.GetString("policyId"),
				Action:               c.GetString("action"),
				RouteId:              c.GetString("routeId"),
				CorrelationId:        cid,
				Metadata:             metadataBytes,
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

			imageResponseMetadata := getCtxImageResponseMetadata(c)
			if imageResponseMetadata != nil {
				enrichedEvent.ImageResponseMetadata = imageResponseMetadata
			}

			pub.Publish(message.Message{
				Type: "event",
				Data: enrichedEvent,
			})
		}()

		if len(fullPath) == 0 {
			telemetry.Incr("bricksllm.proxy.get_middleware.route_does_not_exist", nil, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] route not supported")
			c.Abort()
			return
		}

		kc, settings, err := a.AuthenticateHttpRequest(c.Request, c.Param(xcustom.XProviderIdParam))

		enrichedEvent.Key = kc
		_, ok := err.(notAuthorizedError)
		if ok {
			telemetry.Incr("bricksllm.proxy.get_middleware.authentication_error", nil, 1)
			logError(logWithCid, "error when authenticating http requests", prod, err)
			JSON(c, http.StatusUnauthorized, fmt.Sprintf("[BricksLLM] %v", err))
			c.Abort()
			return
		}

		_, ok = err.(notFoundError)
		if ok {
			telemetry.Incr("bricksllm.proxy.get_middleware.not_found_error", nil, 1)
			logError(logWithCid, "error when authenticating http requests", prod, err)
			JSON(c, http.StatusNotFound, "[BricksLLM] route not found")
			c.Abort()
			return
		}

		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_middleware.authenticate_http_request_error", nil, 1)
			logError(logWithCid, "error when authenticating http requests", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] internal authentication error")
			c.Abort()
			return
		}

		c.Set("key", kc)
		c.Set("settings", settings)

		if kc.ShouldLogResponse {
			blw = &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
			c.Writer = blw
		}

		if len(settings) >= 1 {
			selected := settings[0]

			if selected.CostMap != nil {
				enrichedEvent.CostMap = selected.CostMap
				c.Set("cost_map", selected.CostMap)
			}

			if strings.HasPrefix(fullPath, "/api/providers/azure/openai") {
				if selected != nil && len(selected.Setting["resourceName"]) != 0 {
					c.Set("resourceName", selected.Setting["resourceName"])
				}
			}

			if strings.HasPrefix(fullPath, "/api/providers/bedrock/anthropic") {
				if selected != nil && len(selected.Setting["awsAccessKeyId"]) != 0 {
					c.Set("awsAccessKeyId", selected.Setting["awsAccessKeyId"])
				}

				if selected != nil && len(selected.Setting["awsSecretAccessKey"]) != 0 {
					c.Set("awsSecretAccessKey", selected.Setting["awsSecretAccessKey"])
				}

				if selected != nil && len(selected.Setting["awsRegion"]) != 0 {
					c.Set("awsRegion", selected.Setting["awsRegion"])
				}
			}

			if strings.HasPrefix(fullPath, "/api/providers/vllm") {
				if selected != nil && len(selected.Setting["url"]) != 0 {
					c.Set("vllmUrl", selected.Setting["url"])
				}
			}
		}

		p := pm.GetPolicyByIdFromMemdb(kc.PolicyId)

		c.Set("policyId", kc.PolicyId)

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(logWithCid, "error when reading request body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] error reading request body")
			c.Abort()
			return
		}

		if kc.ShouldLogRequest {
			if len(body) != 0 {
				requestBytes = body
			}

			c.Set("requestBytes", requestBytes)
		}

		if method != http.MethodGet {
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
		}

		if fullPath == "/api/providers/anthropic/v1/complete" {
			logCompletionRequest(logWithCid, body, prod, private)

			cr := &anthropic.CompletionRequest{}
			err = json.Unmarshal(body, cr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling anthropic completion request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling anthropic completion request")
				c.Abort()
				return
			}

			if cr.Metadata != nil {
				userId = cr.Metadata.UserId
			}

			enrichedEvent.Request = cr

			if cr.Stream {
				c.Set("stream", cr.Stream)
			}

			c.Set("model", cr.Model)

			policyInput = cr
		}

		if fullPath == "/api/providers/bedrock/anthropic/v1/complete" {
			logCompletionRequest(logWithCid, body, prod, private)

			cr := &anthropic.CompletionRequest{}
			err = json.Unmarshal(body, cr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling bedrock anthropic completion request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling bedrock anthropic completion request")
				c.Abort()
				return
			}

			if cr.Metadata != nil {
				userId = cr.Metadata.UserId
			}

			enrichedEvent.Request = cr

			if cr.Stream {
				c.Set("stream", cr.Stream)
			}

			c.Set("model", cr.Model)

			policyInput = cr
		}

		if fullPath == "/api/providers/bedrock/anthropic/v1/messages" {
			logCreateMessageRequest(logWithCid, body, prod, private)

			mr := &anthropic.MessagesRequest{}
			err = json.Unmarshal(body, mr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling anthropic messages request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling anthropic messages request")
				c.Abort()
				return
			}

			if mr.Metadata != nil {
				userId = mr.Metadata.UserId
			}

			if mr.Stream {
				c.Set("stream", mr.Stream)
			}

			c.Set("model", mr.Model)

			policyInput = mr
		}

		if fullPath == "/api/providers/anthropic/v1/messages" {
			logCreateMessageRequest(logWithCid, body, prod, private)

			mr := &anthropic.MessagesRequest{}
			err = json.Unmarshal(body, mr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling anthropic messages request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling anthropic messages request")
				c.Abort()
				return
			}

			if mr.Metadata != nil {
				userId = mr.Metadata.UserId
			}

			if mr.Stream {
				c.Set("stream", mr.Stream)
			}

			c.Set("model", mr.Model)

			policyInput = mr
		}

		if strings.HasPrefix(fullPath, "/api/custom/providers/:provider") {
			providerName := c.Param("provider")

			rc := cpm.GetRouteConfigFromMem(providerName, c.Param("wildcard"))
			enrichedEvent.RouteConfig = rc

			cp := cpm.GetCustomProviderFromMem(providerName)
			if cp == nil {
				telemetry.Incr("bricksllm.proxy.get_middleware.provider_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] requested custom provider is not found")
				c.Abort()
				return
			}

			if rc == nil {
				telemetry.Incr("bricksllm.proxy.get_middleware.route_config_not_found", nil, 1)
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
			// 	logError(log, "error when counting tokens for custom provider request", prod, err)
			// }

			// c.Set("promptTokenCount", tks)

			bodyStr := string(body)
			result := gjson.Get(bodyStr, rc.StreamLocation)

			if result.IsBool() {
				c.Set("stream", result.Bool())
			}

			result = gjson.Get(bodyStr, rc.ModelLocation)
			if len(result.Str) != 0 {
				c.Set("model", result.Str)
			}
		}

		if strings.HasPrefix(fullPath, "/api/routes") {
			r := c.Param("route")
			rc := rm.GetRouteFromMemDb(r)

			if rc == nil {
				telemetry.Incr("bricksllm.proxy.get_middleware.route_config_not_found", nil, 1)
				JSON(c, http.StatusNotFound, "[BricksLLM] route config is not found")
				c.Abort()
				return
			}

			c.Set("route_config", rc)
			c.Set("routeId", rc.Id)

			if rc.ShouldRunEmbeddings() {
				er := &goopenai.EmbeddingRequest{}
				err = json.Unmarshal(body, er)
				if err != nil {
					logError(logWithCid, "error when unmarshalling route embedding request", prod, err)
				}

				userId = er.User

				c.Set("model", string(er.Model))

				if rc.CacheConfig != nil && rc.CacheConfig.Enabled {
					c.Set("cache_key", route.ComputeCacheKeyForEmbeddingsRequest(r, er))
				}

				c.Set("encoding_format", string(er.EncodingFormat))

				logEmbeddingRequest(logWithCid, prod, private, er)

				policyInput = er
			}

			if !rc.ShouldRunEmbeddings() {
				ccr := &goopenai.ChatCompletionRequest{}

				err = json.Unmarshal(body, ccr)
				if err != nil {
					logError(logWithCid, "error when unmarshalling route chat completion request", prod, err)
				}

				c.Set("model", ccr.Model)
				userId = ccr.User
				enrichedEvent.Request = ccr

				logRequest(logWithCid, prod, private, ccr)

				if ccr.Stream {
					telemetry.Incr("bricksllm.proxy.get_middleware.streaming_not_allowed", nil, 1)
					JSON(c, http.StatusForbidden, "[BricksLLM] streaming is not allowed")
					c.Abort()
					return
				}

				if rc.CacheConfig != nil && rc.CacheConfig.Enabled {
					c.Set("cache_key", route.ComputeCacheKeyForChatCompletionRequest(r, ccr))
				}

				policyInput = ccr
			}
		}

		if fullPath == "/api/providers/vllm/v1/chat/completions" {
			ccr := &vllm.ChatRequest{}
			err = json.Unmarshal(body, ccr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling vllm chat completions request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling vllm chat completions request")
				c.Abort()
				return
			}

			c.Set("model", ccr.Model)
			userId = ccr.User
			enrichedEvent.Request = ccr

			logVllmChatCompletionRequest(logWithCid, ccr, prod, private)

			if ccr.Stream {
				c.Set("stream", true)
			}

			policyInput = ccr
		}

		if fullPath == "/api/providers/vllm/v1/completions" {
			cr := &vllm.CompletionRequest{}
			err = json.Unmarshal(body, cr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling vllm completions request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling vllm completions request")
				c.Abort()
				return
			}

			c.Set("model", cr.Model)
			userId = cr.User
			enrichedEvent.Request = cr

			logVllmCompletionRequest(logWithCid, cr, prod, private)

			if cr.Stream {
				c.Set("stream", true)
			}

			policyInput = cr
		}

		if fullPath == "/api/providers/deepinfra/v1/chat/completions" {
			ccr := &vllm.ChatRequest{}
			err = json.Unmarshal(body, ccr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling deepinfra chat completions request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling deepinfra chat completions request")
				c.Abort()
				return
			}

			c.Set("model", ccr.Model)
			userId = ccr.User
			enrichedEvent.Request = ccr

			if ccr.Stream {
				c.Set("stream", true)
			}

			logVllmChatCompletionRequest(logWithCid, ccr, prod, private)
			policyInput = ccr
		}

		if fullPath == "/api/providers/deepinfra/v1/completions" {
			cr := &vllm.CompletionRequest{}
			err = json.Unmarshal(body, cr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling deepinfra completions request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling deepinfra completions request")
				c.Abort()
				return
			}

			c.Set("model", cr.Model)
			userId = cr.User
			enrichedEvent.Request = cr

			if cr.Stream {
				c.Set("stream", true)
			}

			logVllmCompletionRequest(logWithCid, cr, prod, private)
			policyInput = cr
		}

		if fullPath == "/api/providers/deepinfra/v1/embeddings" {
			er := &goopenai.EmbeddingRequest{}
			err = json.Unmarshal(body, er)
			if err != nil {
				logError(logWithCid, "error when unmarshalling deepinfra embeddings request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling deepinfra embeddings request")
				c.Abort()
				return
			}

			userId = er.User
			enrichedEvent.Request = er

			c.Set("model", string(er.Model))

			logEmbeddingRequest(logWithCid, prod, private, er)
			policyInput = er
		}

		if fullPath == "/api/providers/azure/openai/deployments/:deployment_id/chat/completions" {
			ccr := &goopenai.ChatCompletionRequest{}
			err = json.Unmarshal(body, ccr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling azure openai chat completion request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling azure openai chat completion request")
				c.Abort()
				return
			}

			userId = ccr.User
			enrichedEvent.Request = ccr
			c.Set("model", ccr.Model)

			logRequest(logWithCid, prod, private, ccr)

			if ccr.Stream {
				c.Set("stream", true)
			}

			policyInput = ccr
		}

		if fullPath == "/api/providers/azure/openai/deployments/:deployment_id/completions" {
			cr := &goopenai.CompletionRequest{}
			err = json.Unmarshal(body, cr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling azure openai completions request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling azure openai completions request")
				c.Abort()
				return
			}

			userId = cr.User
			enrichedEvent.Request = cr
			c.Set("model", cr.Model)

			logAzureCompletionsRequest(logWithCid, prod, private, cr)

			if cr.Stream {
				c.Set("stream", true)
			}

			policyInput = cr
		}

		if fullPath == "/api/providers/azure/openai/deployments/:deployment_id/embeddings" {
			er := &goopenai.EmbeddingRequest{}
			err = json.Unmarshal(body, er)
			if err != nil {
				logError(logWithCid, "error when unmarshalling azure openai embedding request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling azure openai embedding request")
				c.Abort()
				return
			}

			userId = er.User

			c.Set("model", "ada")
			c.Set("encoding_format", string(er.EncodingFormat))

			logEmbeddingRequest(logWithCid, prod, private, er)

			policyInput = er
		}

		if fullPath == "/api/providers/openai/v1/chat/completions" {
			ccr := &goopenai.ChatCompletionRequest{}
			// this is a hack around an open issue in go-openai.
			// https://github.com/sashabaranov/go-openai/issues/884
			cleaned, err := sjson.Delete(string(body), "response_format.json_schema")
			if err != nil {
				logWithCid.Warn("removing response_format.json_schema", zap.Error(err))
			}
			err = json.Unmarshal([]byte(cleaned), ccr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling chat completion request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling chat completion request")
				c.Abort()
				return
			}

			userId = ccr.User

			enrichedEvent.Request = ccr

			c.Set("model", ccr.Model)

			logRequest(logWithCid, prod, private, ccr)

			if ccr.Stream {
				c.Set("stream", true)
			}

			policyInput = ccr
		}

		if strings.HasPrefix(fullPath, "/api/providers/openai/v1/responses") {
			responsesReq := &openai.ResponseRequest{}
			err = json.Unmarshal(body, responsesReq)
			if err != nil {
				logError(logWithCid, "error when unmarshalling openai responses request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling openai responses request")
				c.Abort()
				return
			}

			ginCtxSetResponsesRequest(c, responsesReq)

			if gopointer.ToValueOrDefault(responsesReq.Background, false) {
				telemetry.Incr("bricksllm.proxy.get_middleware.background_not_allowed", nil, 1)
				JSON(c, http.StatusForbidden, "[BricksLLM] background is not allowed")
				c.Abort()
				return
			}

			hasNotAllowedTools := false
			isCreateContainerTool := false
			var containerMemLimit string
			for _, tool := range responsesReq.Tools {
				if _, ok := openai.AllowedToolsSet[tool.Type]; !ok {
					hasNotAllowedTools = true
					break
				}

				if container := tool.GetContainerAsResponseRequestToolContainer(); container != nil {
					isCreateContainerTool = true
					containerMemLimit = container.GetMemoryLimit()
				}
			}

			if hasNotAllowedTools {
				telemetry.Incr("bricksllm.proxy.get_middleware.tool_not_allowed", nil, 1)
				JSON(c, http.StatusForbidden, "[BricksLLM] one of the tools is not allowed")
				c.Abort()
				return
			}

			if isCreateContainerTool {
				_, ok := openai.OpenAiCodeInterpreterContainerCost[containerMemLimit]
				if !ok {
					telemetry.Incr("bricksllm.proxy.get_middleware.container_memory_limit_not_allowed", nil, 1)
					JSON(c, http.StatusForbidden, "[BricksLLM] container memory limit is not allowed")
					c.Abort()
					return
				}
			}

			userId = gopointer.ToValueOrDefault(responsesReq.SafetyIdentifier, "")
			enrichedEvent.Request = responsesReq
			c.Set("model", gopointer.ToValueOrDefault(responsesReq.Model, ""))

			logResponsesRequest(logWithCid, prod, private, responsesReq)

			if gopointer.ToValueOrDefault(responsesReq.Stream, false) {
				c.Set("stream", true)
			}

			policyInput = responsesReq
		}

		if fullPath == "/api/providers/openai/v1/embeddings" {
			er := &goopenai.EmbeddingRequest{}
			err = json.Unmarshal(body, er)
			if err != nil {
				logError(logWithCid, "error when unmarshalling embedding request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling embedding request")
				c.Abort()
				return
			}

			userId = er.User

			c.Set("model", string(er.Model))
			c.Set("encoding_format", string(er.EncodingFormat))

			logEmbeddingRequest(logWithCid, prod, private, er)

			policyInput = er
		}

		if fullPath == "/api/providers/openai/v1/images/generations" && method == http.MethodPost {
			ir := &goopenai.ImageRequest{}
			err := json.Unmarshal(body, ir)
			if err != nil {
				logError(logWithCid, "error when unmarshalling create image request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling create image request")
				c.Abort()
				return
			}
			enrichedEvent.Request = ir

			c.Set("model", ir.Model)

			if len(ir.Model) == 0 {
				c.Set("model", "dall-e-2")
			}

			c.Set("model", ir.Model)
			logCreateImageRequest(logWithCid, ir, prod, private)
		}

		if fullPath == "/api/providers/openai/v1/images/edits" && method == http.MethodPost {
			ier := &goopenai.ImageEditRequest{}
			err := json.Unmarshal(body, ier)
			if err != nil {
				logError(logWithCid, "error when unmarshalling edit image request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling edit image request")
				c.Abort()
				return
			}
			enrichedEvent.Request = ier

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

			logEditImageRequest(logWithCid, prompt, model, n, size, responseFormat, user, prod, private)
		}

		if fullPath == "/api/providers/openai/v1/images/variations" && method == http.MethodPost {
			ir := &goopenai.ImageVariRequest{}
			err := json.Unmarshal(body, ir)
			if err != nil {
				logError(logWithCid, "error when unmarshalling image variations request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling image variations request")
				c.Abort()
				return
			}
			enrichedEvent.Request = ir

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

			logImageVariationsRequest(logWithCid, model, n, size, responseFormat, user, prod)
		}

		if fullPath == "/api/providers/openai/v1/audio/speech" && method == http.MethodPost {
			sr := &goopenai.CreateSpeechRequest{}
			err := json.Unmarshal(body, sr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling create speech request", prod, err)
				JSON(c, http.StatusBadRequest, "[BricksLLM] error when unmarshalling create speech request")
				c.Abort()
				return
			}

			enrichedEvent.Request = sr

			c.Set("model", string(sr.Model))

			logCreateSpeechRequest(logWithCid, sr, prod, private)
		}

		if fullPath == "/api/providers/openai/v1/audio/transcriptions" && method == http.MethodPost {
			model := c.PostForm("model")
			language := c.PostForm("language")
			prompt := c.PostForm("prompt")
			responseFormat := c.PostForm("response_format")
			temperature := c.PostForm("temperature")

			c.Set("model", model)

			converted, _ := strconv.ParseFloat(temperature, 64)
			logCreateTranscriptionRequest(logWithCid, model, language, prompt, responseFormat, converted, prod, private)
		}

		if fullPath == "/api/providers/openai/v1/audio/translations" && method == http.MethodPost {
			model := c.PostForm("model")
			prompt := c.PostForm("prompt")
			responseFormat := c.PostForm("response_format")
			temperature := c.PostForm("temperature")

			c.Set("model", model)

			converted, _ := strconv.ParseFloat(temperature, 64)
			logCreateTranslationRequest(logWithCid, model, prompt, responseFormat, converted, prod, private)
		}

		if strings.HasPrefix(fullPath, "/api/providers/openai/v1/videos") && method == http.MethodPost {
			model := c.PostForm("model")
			if model == "" {
				vr := &openai.VideoRequest{}
				err := json.Unmarshal(body, vr)
				if err != nil {
					logError(logWithCid, "error when unmarshalling video request", prod, err)
				}
				enrichedEvent.Request = vr
				model = vr.Model
			}
			c.Set("model", model)
		}

		if len(kc.AllowedPaths) != 0 && !containsPath(kc.AllowedPaths, fullPath, method) {
			telemetry.Incr("bricksllm.proxy.get_middleware.path_not_allowed", nil, 1)
			JSON(c, http.StatusForbidden, "[BricksLLM] path is not allowed")
			c.Abort()
			return
		}

		model := c.GetString("model")
		if !isModelAllowed(model, settings) {
			telemetry.Incr("bricksllm.proxy.get_middleware.model_not_allowed", nil, 1)
			JSON(c, http.StatusForbidden, "[BricksLLM] model is not allowed")
			c.Abort()
			return
		}
		if !isModelSupported(fullPath, model) {
			telemetry.Incr("bricksllm.proxy.get_middleware.model_not_supported", nil, 1)
			JSON(c, http.StatusBadRequest, "[BricksLLM] model is not supported")
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

		if fullPath == "/api/providers/openai/v1/assistants" && method == http.MethodPost {
			logCreateAssistantRequest(logWithCid, body, prod, private)

			ar := &goopenai.AssistantRequest{}

			err = json.Unmarshal(body, ar)
			if err != nil {
				logError(logWithCid, "error when unmarshalling assistant request", prod, err)
			}

			if err == nil {
				c.Set("model", ar.Model)

				policyInput = ar
			}
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id" && method == http.MethodGet {
			logRetrieveAssistantRequest(logWithCid, prod, aid)
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id" && method == http.MethodPost {
			logModifyAssistantRequest(logWithCid, body, prod, private, aid)
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id" && method == http.MethodDelete {
			logDeleteAssistantRequest(logWithCid, prod, aid)
		}

		if fullPath == "/api/providers/openai/v1/assistants" && method == http.MethodGet {
			logListAssistantsRequest(logWithCid, prod)
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id/files" && method == http.MethodPost {
			logCreateAssistantFileRequest(logWithCid, body, prod, aid)
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && method == http.MethodGet {
			logRetrieveAssistantFileRequest(logWithCid, prod, fid, aid)
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && method == http.MethodDelete {
			logDeleteAssistantFileRequest(logWithCid, prod, fid, aid)
		}

		if fullPath == "/api/providers/openai/v1/assistants/:assistant_id/files" && method == http.MethodGet {
			logListAssistantFilesRequest(logWithCid, prod, aid, qm)
		}

		if fullPath == "/api/providers/openai/v1/threads" && method == http.MethodPost {
			logCreateThreadRequest(logWithCid, body, prod, private)

			tr := &openai.ThreadRequest{}

			err = json.Unmarshal(body, tr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling create thread request", prod, err)
			}

			if err == nil {
				policyInput = tr
			}
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id" && method == http.MethodGet {
			logRetrieveThreadRequest(logWithCid, prod, tid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id" && method == http.MethodPost {
			logModifyThreadRequest(logWithCid, body, prod, tid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id" && method == http.MethodDelete {
			logDeleteThreadRequest(logWithCid, prod, tid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/messages" && method == http.MethodPost {
			logCreateMessageRequest(logWithCid, body, prod, private)

			mr := &openai.MessageRequest{}
			err := json.Unmarshal(body, mr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling create message request", prod, err)
			}

			if err == nil {
				policyInput = mr
			}
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && method == http.MethodGet {
			logRetrieveMessageRequest(logWithCid, prod, mid, tid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && method == http.MethodPost {
			logModifyMessageRequest(logWithCid, body, prod, private, tid, mid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/messages" && method == http.MethodGet {
			logListMessagesRequest(logWithCid, prod, aid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id" && method == http.MethodGet {
			logRetrieveMessageFileRequest(logWithCid, prod, mid, tid, fid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files" && method == http.MethodGet {
			logListMessageFilesRequest(logWithCid, prod, tid, mid, qm)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs" && method == http.MethodPost {
			logCreateRunRequest(logWithCid, body, prod, private)

			rr := &goopenai.RunRequest{}
			err := json.Unmarshal(body, rr)
			if err != nil {
				logError(logWithCid, "error when unmarshalling create run request", prod, err)
			}

			if err == nil {
				c.Set("model", rr.Model)
				policyInput = rr
			}
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && method == http.MethodGet {
			logRetrieveRunRequest(logWithCid, prod, tid, rid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && method == http.MethodPost {
			logModifyRunRequest(logWithCid, body, prod, tid, rid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs" && method == http.MethodGet {
			logListRunsRequest(logWithCid, prod, tid, qm)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs" && method == http.MethodPost {
			logSubmitToolOutputsRequest(logWithCid, body, prod, tid, rid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel" && method == http.MethodPost {
			logCancelARunRequest(logWithCid, prod, tid, rid)
		}

		if fullPath == "/api/providers/openai/v1/threads/runs" && method == http.MethodPost {
			logCreateThreadAndRunRequest(logWithCid, body, prod, private)

			r := &openai.CreateThreadAndRunRequest{}
			err := json.Unmarshal(body, r)
			if err != nil {
				logError(logWithCid, "error when unmarshalling create thread and run request", prod, err)
			}

			if err == nil {
				c.Set("model", r.Model)
				policyInput = r
			}
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id" && method == http.MethodGet {
			logRetrieveRunStepRequest(logWithCid, prod, tid, rid, sid)
		}

		if fullPath == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps" && method == http.MethodGet {
			logListRunStepsRequest(logWithCid, prod, tid, rid, qm)
		}

		if fullPath == "/api/providers/openai/v1/moderations" && method == http.MethodPost {
			logCreateModerationRequest(logWithCid, body, prod, private)
		}

		if fullPath == "/api/providers/openai/v1/models/:model" && method == http.MethodGet {
			logRetrieveModelRequest(logWithCid, prod, md)
		}

		if fullPath == "/api/providers/openai/v1/models/:model" && method == http.MethodDelete {
			logDeleteModelRequest(logWithCid, prod, md)
		}

		if fullPath == "/api/providers/openai/v1/files" && method == http.MethodGet {
			logListFilesRequest(logWithCid, prod, qm)
		}

		if fullPath == "/api/providers/openai/v1/files" && method == http.MethodPost {
			purpose := c.PostForm("purpose")
			logUploadFileRequest(logWithCid, prod, purpose)
		}

		if fullPath == "/api/providers/openai/v1/files/:file_id" && method == http.MethodDelete {
			logDeleteFileRequest(logWithCid, prod, fid)
		}

		if fullPath == "/api/providers/openai/v1/files/:file_id" && method == http.MethodGet {
			logRetrieveFileRequest(logWithCid, prod, fid)
		}

		if fullPath == "/api/providers/openai/v1/files/:file_id/content" && method == http.MethodGet {
			logRetrieveFileContentRequest(logWithCid, prod, fid)
		}

		if ac.GetAccessStatus(kc.KeyId) {
			telemetry.Incr("bricksllm.proxy.get_middleware.rate_limited", nil, 1)
			JSON(c, http.StatusTooManyRequests, "[BricksLLM] You've reached your AI usage limit for this course. To continue, please contact your instructor or TA.")
			c.Abort()
			return
		}

		if len(userId) != 0 {
			c.Set("userId", userId)
			us, err := um.GetUsers(kc.Tags, nil, []string{userId}, 0, 0)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_middleware.get_users_error", nil, 1)
				logError(logWithCid, "error when getting users", prod, err)
			}

			if len(us) == 1 {
				if us[0].Revoked {
					telemetry.Incr("bricksllm.proxy.get_middleware.user_revoked", nil, 1)
					JSON(c, http.StatusUnauthorized, fmt.Sprintf("[BricksLLM] user is revoked: %s", userId))
					c.Abort()
					return
				}

				model := c.GetString("model")
				if len(us[0].AllowedModels) != 0 && !contains(us[0].AllowedModels, model) {
					telemetry.Incr("bricksllm.proxy.get_middleware.user_requested_model_not_allowed", nil, 1)
					JSON(c, http.StatusForbidden, fmt.Sprintf("[BricksLLM] model: %s forbidden for user: %s", model, userId))
					c.Abort()
					return
				}

				if len(us[0].AllowedPaths) != 0 && !containsPath(us[0].AllowedPaths, fullPath, method) {
					telemetry.Incr("bricksllm.proxy.get_middleware.user_requested_path_not_allowed", nil, 1)
					JSON(c, http.StatusForbidden, fmt.Sprintf("[BricksLLM] path: %s forbidden for user: %s", fullPath, userId))
					c.Abort()
					return
				}

				if uac.GetAccessStatus(us[0].Id) {
					telemetry.Incr("bricksllm.proxy.get_middleware.user_rate_limited", nil, 1)
					JSON(c, http.StatusTooManyRequests, fmt.Sprintf("[BricksLLM] You've reached your AI usage limit for this course for user: %s. To continue, please contact your instructor or TA.", userId))
					c.Abort()
					return
				}
			}

			if len(us) > 1 {
				telemetry.Incr("bricksllm.proxy.get_middleware.get_multiple_users_error", nil, 1)
			}
		}

		if p != nil {
			c.Set("policyId", p.Id)
		}

		if p != nil && policyInput != nil {
			err := p.Filter(client, policyInput, scanner, cd, logWithCid)
			if err == nil {
				c.Set("action", "allowed")
			}

			if err != nil {
				_, ok := err.(blockedError)
				if ok {
					c.Set("action", "blocked")
					telemetry.Incr("bricksllm.proxy.get_middleware.request_blocked", nil, 1)
					JSON(c, http.StatusForbidden, "[BricksLLM] request blocked")
					c.Abort()
					return
				}

				_, ok = err.(warnedError)
				if ok {
					c.Set("action", "warned")
				}

				_, ok = err.(redactedError)
				if ok {
					c.Set("action", "redacted")
				}

				logError(logWithCid, "error when filtering a request", prod, err)
			}

			data, err := json.Marshal(policyInput)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewReader(data))

				if kc.ShouldLogRequest {
					requestBytes = data
				}
			}
		}

		c.Next()

		if kc.ShouldLogResponse {
			if c.GetBool("stream") {
				streamingResponse, ok := c.Get("streaming_response")

				if ok {
					bs, _ := streamingResponse.([]byte)
					if len(bs) != 0 {
						streamingData := &StreamingData{
							Data: bs,
						}

						jbs, err := json.Marshal(streamingData)
						if err != nil {
							telemetry.Incr("bricksllm.proxy.get_middleware.streaming_data_json_marshal_error", nil, 1)
							logError(logWithCid, "error when marshalling streaming data into json", prod, err)
						}

						responseBytes = jbs
					}
				}
			}

			if !c.GetBool("stream") && blw != nil {
				responseData := blw.body.Bytes()

				if len(responseData) != 0 {
					responseBytes = responseData
				}
			}
		}
	}
}

type StreamingData struct {
	Data []byte `json:"data"`
}

func contains(arr []string, target string) bool {
	return slices.Contains(arr, target)
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

var openaiModels = map[string]struct{}{}
var anthropicModels = map[string]struct{}{}
var azureOpenAIModels = map[string]struct{}{}
var deepinfraModels = map[string]struct{}{}

func init() {
	// openai models
	initByCostMap(openaiModels, openai.OpenAiPerThousandTokenCost)
	// anthropic models
	initByCostMap(anthropicModels, anthropic.AnthropicPerMillionTokenCost)
	// azure openai models
	initByCostMap(azureOpenAIModels, azure.AzureOpenAiPerThousandTokenCost)
	// deepinfra models
	initByCostMap(deepinfraModels, deepinfra.DeepinfraPerMillionTokenCost)
}

func initByCostMap(target map[string]struct{}, source map[string]map[string]float64) {
	for _, m := range source {
		for k, _ := range m {
			target[strings.ToLower(k)] = struct{}{}
		}
	}
}

func isModelSupported(path, model string) bool {
	if len(model) == 0 {
		return true
	}

	targetModel := strings.ToLower(model)
	if strings.HasPrefix(path, "/api/providers/anthropic") {
		targetModel = anthropic.SelectModel(targetModel)
	}
	models := modelsMapByPath(path)
	if models == nil {
		return true
	}
	if _, ok := models[targetModel]; ok {
		return true
	}
	return false
}

func modelsMapByPath(path string) map[string]struct{} {
	if strings.HasPrefix(path, "/api/providers/openai") {
		return openaiModels
	}
	if strings.HasPrefix(path, "/api/providers/anthropic") {
		return anthropicModels
	}
	if strings.HasPrefix(path, "/api/providers/azure/openai") {
		return azureOpenAIModels
	}
	if strings.HasPrefix(path, "/api/providers/deepinfra") {
		return deepinfraModels
	}
	return nil
}
