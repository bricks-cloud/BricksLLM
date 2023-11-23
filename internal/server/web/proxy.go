package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	correlationId string = "correlationId"
)

type ProxyServer struct {
	server *http.Server
	log    *zap.Logger
}

type recorder interface {
	RecordKeySpend(keyId string, model string, micros int64, costLimitUnit key.TimeUnit) error
	RecordEvent(e *event.Event) error
}

func NewProxyServer(log *zap.Logger, mode, privacyMode string, m KeyManager, psm ProviderSettingsManager, ks keyStorage, kms keyMemStorage, e estimator, v validator, r recorder, credential string, enc encrypter, rlm rateLimitManager, timeOut time.Duration) (*ProxyServer, error) {
	router := gin.New()
	prod := mode == "production"
	private := privacyMode == "strict"

	router.Use(getMiddleware(kms, prod, private, e, v, ks, log, enc, rlm, r, "proxy"))

	client := http.Client{}

	router.POST("/api/health", getGetHealthCheckHandler())
	router.POST("/api/providers/openai/v1/chat/completions", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	router.POST("/api/providers/openai/v1/embeddings", getEmbeddingHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))

	// router.GET("/api/providers/openai/v1/assistants", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.POST("/api/providers/openai/v1/assistants", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.GET("/api/providers/openai/v1/assistants/:assistant_id", getEmbeddingHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.POST("/api/providers/openai/v1/assistants/:assistant_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.DELETE("/api/providers/openai/v1/assistants/:assistant_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))

	// router.GET("/api/providers/openai/v1/assistants", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.POST("/api/providers/openai/v1/assistants", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.GET("/api/providers/openai/v1/assistants/:assistant_id", getEmbeddingHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.POST("/api/providers/openai/v1/assistants/:assistant_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.DELETE("/api/providers/openai/v1/assistants/:assistant_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))

	// router.POST("/api/providers/openai/v1/threads", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.GET("/api/providers/openai/v1/threads/:thread_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.POST("/api/providers/openai/v1/threads/:thread_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.DELETE("/api/providers/openai/v1/threads/:thread_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))

	// router.POST("/api/providers/openai/v1/threads/:thread_id/messages", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.POST("/api/providers/openai/v1/threads/:thread_id/messages/:message_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.GET("/api/providers/openai/v1/threads/:thread_id/messages", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))

	// router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))
	// router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files", getChatCompletionHandler(r, prod, private, psm, client, kms, log, enc, e, timeOut))

	srv := &http.Server{
		Addr:    ":8002",
		Handler: router,
	}

	return &ProxyServer{
		log:    log,
		server: srv,
	}, nil
}

func createAuthenticatedHttpRequest(ctx context.Context, c *gin.Context, log *zap.Logger, prod bool, psm ProviderSettingsManager, functionId, cid, settingId string, targetUrl string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetUrl, c.Request.Body)
	if err != nil {
		logError(log, "error when creating openai http request", prod, cid, err)
		JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
		return nil, errors.New("error creating http request")
	}

	req.Header.Set("Content-Type", "application/json")
	setting, err := psm.GetSetting(settingId)
	if err != nil {
		stats.Incr(fmt.Sprintf("bricksllm.web.%s.provider_setting_not_found", functionId), nil, 1)

		logError(log, "openai api key is not set", prod, cid, err)
		JSON(c, http.StatusInternalServerError, "[BricksLLM] openai api key is not set")
		return nil, errors.New("open ai key is not set")
	}

	key, ok := setting.Setting["apikey"]
	if !ok || len(key) == 0 {
		stats.Incr(fmt.Sprintf("bricksllm.web.%s.provider_setting_api_key_not_found", functionId), nil, 1)

		logError(log, "openai api key is not found in setting", prod, cid, err)
		JSON(c, http.StatusInternalServerError, "[BricksLLM] openai api key is not found in setting")
		return nil, errors.New("open ai key is not found in setting")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	return req, nil
}

func getPassThroughHandler(r recorder, prod, private bool, psm ProviderSettingsManager, client http.Client, kms keyMemStorage, log *zap.Logger, enc encrypter, e estimator, timeOut time.Duration, targetUrl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.web.get_pass_through_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		raw, exists := c.Get("key")
		kc, ok := raw.(*key.ResponseKey)
		if !exists || !ok {
			stats.Incr("bricksllm.web.get_pass_through_handler.api_key_not_registered", nil, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
			return
		}

		id := c.GetString(correlationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := createAuthenticatedHttpRequest(ctx, c, log, prod, psm, "get_pass_through_handler", id, kc.SettingId, targetUrl)
		if err != nil {
			return
		}

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.web.get_pass_through_handler.http_client_error", nil, 1)

			logError(log, "error when sending embedding request to openai", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send embedding request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Now().Sub(start)
		stats.Timing("bricksllm.web.get_pass_through_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai embedding response body")
			return
		}

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.web.get_pass_through_handler.success", nil, 1)
			stats.Timing("bricksllm.web.get_pass_through_handler.success_latency", dur, nil, 1)
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.web.get_pass_through_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.web.get_pass_through_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai pass through error response body", prod, id, err)
			}

			logOpenAiError(log, prod, id, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

// EmbeddingResponse is the response from a Create embeddings request.
type EmbeddingResponse struct {
	Object string               `json:"object"`
	Data   []goopenai.Embedding `json:"data"`
	Model  string               `json:"model"`
	Usage  goopenai.Usage       `json:"usage"`
}

// EmbeddingResponse is the response from a Create embeddings request.
type EmbeddingResponseBase64 struct {
	Object string                     `json:"object"`
	Data   []goopenai.Base64Embedding `json:"data"`
	Model  string                     `json:"model"`
	Usage  goopenai.Usage             `json:"usage"`
}

func getEmbeddingHandler(r recorder, prod, private bool, psm ProviderSettingsManager, client http.Client, kms keyMemStorage, log *zap.Logger, enc encrypter, e estimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.web.get_embedding_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		raw, exists := c.Get("key")
		kc, ok := raw.(*key.ResponseKey)
		if !exists || !ok {
			stats.Incr("bricksllm.web.get_embedding_handler.api_key_not_registered", nil, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
			return
		}

		id := c.GetString(correlationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := createAuthenticatedHttpRequest(ctx, c, log, prod, psm, "get_embedding_handler", id, kc.SettingId, "https://api.openai.com/v1/embeddings")
		if err != nil {
			return
		}

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.web.get_embedding_handler.http_client_error", nil, 1)

			logError(log, "error when sending embedding request to openai", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send embedding request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Now().Sub(start)
		stats.Timing("bricksllm.web.get_embedding_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai embedding response body")
			return
		}

		var cost float64 = 0
		chatRes := &EmbeddingResponse{}
		base64ChatRes := &EmbeddingResponseBase64{}
		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.web.get_embedding_handler.success", nil, 1)
			stats.Timing("bricksllm.web.get_embedding_handler.success_latency", dur, nil, 1)

			format := c.GetString("encoding_format")

			if format == "base64" {
				err = json.Unmarshal(bytes, base64ChatRes)
				if err != nil {
					logError(log, "error when unmarshalling openai base64 embedding response body", prod, id, err)
				}
			}

			if format != "base64" {
				err = json.Unmarshal(bytes, chatRes)
				if err != nil {
					logError(log, "error when unmarshalling openai embedding response body", prod, id, err)
				}
			}

			model := c.GetString("model")

			totalTokens := 0
			if err == nil {
				if format == "base64" {
					logBase64EmbeddingResponse(log, prod, private, id, base64ChatRes)
					totalTokens = base64ChatRes.Usage.TotalTokens
				}

				if format != "base64" {
					logEmbeddingResponse(log, prod, private, id, chatRes)
					totalTokens = chatRes.Usage.TotalTokens
				}

				cost, err = e.EstimateEmbeddingsInputCost(model, totalTokens)
				if err != nil {
					stats.Incr("bricksllm.web.get_embedding_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost for embedding", prod, id, err)
				}

				micros := int64(cost * 1000000)
				err = r.RecordKeySpend(kc.KeyId, model, micros, kc.CostLimitInUsdUnit)
				if err != nil {
					stats.Incr("bricksllm.web.get_embedding_handler.record_key_spend_error", nil, 1)
					logError(log, "error when recording openai spend for embedding", prod, id, err)
				}
			}
		}

		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", chatRes.Usage.PromptTokens)
		c.Set("completionTokenCount", chatRes.Usage.CompletionTokens)

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.web.get_embedding_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.web.get_embedding_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai embedding error response body", prod, id, err)
			}

			logOpenAiError(log, prod, id, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func getChatCompletionHandler(r recorder, prod, private bool, psm ProviderSettingsManager, client http.Client, kms keyMemStorage, log *zap.Logger, enc encrypter, e estimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.web.get_chat_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", c.Request.Body)
		id := c.GetString(correlationId)
		if err != nil {
			logError(log, "error when creating openai http request", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		raw, exists := c.Get("key")
		kc, ok := raw.(*key.ResponseKey)
		if !exists || !ok {
			stats.Incr("bricksllm.web.get_chat_completion_handler.api_key_not_registered", nil, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
			return
		}

		req.Header.Set("Content-Type", "application/json")

		setting, err := psm.GetSetting(kc.SettingId)
		if err != nil {
			stats.Incr("bricksllm.web.get_chat_completion_handler.provider_setting_not_found", nil, 1)

			logError(log, "openai api key is not set", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] openai api key is not set")
			return
		}

		key, ok := setting.Setting["apikey"]
		if !ok || len(key) == 0 {
			stats.Incr("bricksllm.web.get_chat_completion_handler.provider_setting_api_key_not_found", nil, 1)

			logError(log, "openai api key is not found in setting", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] openai api key is not found in setting")
			return
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.web.get_chat_completion_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}
		defer res.Body.Close()
		dur := time.Now().Sub(start)
		stats.Timing("bricksllm.web.get_chat_completion_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai http chat completion response body", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		var cost float64 = 0
		chatRes := &goopenai.ChatCompletionResponse{}
		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.web.get_chat_completion_handler.success", nil, 1)
			stats.Timing("bricksllm.web.get_chat_completion_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling openai http chat completion response body", prod, id, err)
			}

			model := c.GetString("model")
			if err == nil {
				logChatCompletionResponse(log, prod, private, id, chatRes)
				cost, err = e.EstimateTotalCost(model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
				if err != nil {
					stats.Incr("bricksllm.web.get_chat_completion_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost", prod, id, err)
				}

				micros := int64(cost * 1000000)
				err = r.RecordKeySpend(kc.KeyId, model, micros, kc.CostLimitInUsdUnit)
				if err != nil {
					stats.Incr("bricksllm.web.get_chat_completion_handler.record_key_spend_error", nil, 1)
					logError(log, "error when recording openai spend", prod, id, err)
				}
			}
		}

		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", chatRes.Usage.PromptTokens)
		c.Set("completionTokenCount", chatRes.Usage.CompletionTokens)

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.web.get_chat_completion_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.web.get_chat_completion_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai http error response body", prod, id, err)
			}

			logOpenAiError(log, prod, id, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func (ps *ProxyServer) Run() {
	go func() {
		ps.log.Info("proxy server listening at 8002")
		ps.log.Info("PORT 8002 | POST  | /api/providers/openai/v1/chat/completions is ready for forwarding chat completion requests to openai")
		ps.log.Info("PORT 8002 | POST  | /api/providers/openai/v1/embeddings is ready for forwarding embeddings requests to openai")

		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ps.log.Sugar().Fatalf("error proxy server listening: %v", err)
			return
		}
	}()
}

func logEmbeddingResponse(log *zap.Logger, prod, private bool, cid string, r *EmbeddingResponse) {
	if prod {
		log.Info("openai embeddings response",
			zap.Time("createdAt", time.Now()),
			zap.String(correlationId, cid),
			zap.Object("response", zapcore.ObjectMarshalerFunc(
				func(enc zapcore.ObjectEncoder) error {
					enc.AddString("object", r.Object)
					enc.AddString("model", r.Model)
					enc.AddArray("data", zapcore.ArrayMarshalerFunc(
						func(enc zapcore.ArrayEncoder) error {
							for _, d := range r.Data {
								enc.AppendObject(zapcore.ObjectMarshalerFunc(
									func(enc zapcore.ObjectEncoder) error {
										enc.AddInt("index", d.Index)
										enc.AddString("object", d.Object)
										if !private {
											enc.AddArray("embedding", zapcore.ArrayMarshalerFunc(
												func(enc zapcore.ArrayEncoder) error {
													for _, e := range d.Embedding {
														enc.AppendFloat32(e)
													}
													return nil
												}))
										}

										return nil
									},
								))
							}
							return nil
						},
					))

					enc.AddObject("usage", zapcore.ObjectMarshalerFunc(
						func(enc zapcore.ObjectEncoder) error {
							enc.AddInt("prompt_tokens", r.Usage.PromptTokens)
							enc.AddInt("completion_tokens", r.Usage.CompletionTokens)
							enc.AddInt("total_tokens", r.Usage.TotalTokens)
							return nil
						},
					))
					return nil
				},
			)),
		)
	}
}

func logBase64EmbeddingResponse(log *zap.Logger, prod, private bool, cid string, r *EmbeddingResponseBase64) {
	if prod {
		log.Info("openai embeddings response",
			zap.Time("createdAt", time.Now()),
			zap.String(correlationId, cid),
			zap.Object("response", zapcore.ObjectMarshalerFunc(
				func(enc zapcore.ObjectEncoder) error {
					enc.AddString("object", r.Object)
					enc.AddString("model", r.Model)
					enc.AddArray("data", zapcore.ArrayMarshalerFunc(
						func(enc zapcore.ArrayEncoder) error {
							for _, d := range r.Data {
								enc.AppendObject(zapcore.ObjectMarshalerFunc(
									func(enc zapcore.ObjectEncoder) error {
										enc.AddInt("index", d.Index)
										enc.AddString("object", d.Object)
										if !private {
											enc.AddString("embedding", string(d.Embedding))
										}

										return nil
									},
								))
							}
							return nil
						},
					))

					enc.AddObject("usage", zapcore.ObjectMarshalerFunc(
						func(enc zapcore.ObjectEncoder) error {
							enc.AddInt("prompt_tokens", r.Usage.PromptTokens)
							enc.AddInt("completion_tokens", r.Usage.CompletionTokens)
							enc.AddInt("total_tokens", r.Usage.TotalTokens)
							return nil
						},
					))
					return nil
				},
			)),
		)
	}
}

func logChatCompletionResponse(log *zap.Logger, prod, private bool, cid string, r *goopenai.ChatCompletionResponse) {
	if prod {
		log.Info("openai chat completion response",
			zap.Time("createdAt", time.Now()),
			zap.String(correlationId, cid),
			zap.Object("response", zapcore.ObjectMarshalerFunc(
				func(enc zapcore.ObjectEncoder) error {
					enc.AddString("id", r.ID)
					enc.AddString("object", r.Object)
					enc.AddInt64("created", r.Created)
					enc.AddString("model", r.Model)
					enc.AddArray("choices", zapcore.ArrayMarshalerFunc(
						func(enc zapcore.ArrayEncoder) error {
							for _, c := range r.Choices {
								enc.AppendObject(zapcore.ObjectMarshalerFunc(
									func(enc zapcore.ObjectEncoder) error {
										enc.AddInt("index", c.Index)
										enc.AddObject("message", zapcore.ObjectMarshalerFunc(
											func(enc zapcore.ObjectEncoder) error {
												enc.AddString("role", c.Message.Role)
												if !private {
													enc.AddString("content", c.Message.Content)
												}
												return nil
											},
										))

										enc.AddString("finish_reason", string(c.FinishReason))
										return nil
									},
								))
							}
							return nil
						},
					))

					enc.AddObject("usage", zapcore.ObjectMarshalerFunc(
						func(enc zapcore.ObjectEncoder) error {
							enc.AddInt("prompt_tokens", r.Usage.PromptTokens)
							enc.AddInt("completion_tokens", r.Usage.CompletionTokens)
							enc.AddInt("total_tokens", r.Usage.TotalTokens)
							return nil
						},
					))
					return nil
				},
			)),
		)
	}
}

func logEmbeddingRequest(log *zap.Logger, prod, private bool, id string, r *goopenai.EmbeddingRequest) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, id),
			zap.String("model", r.Model.String()),
			zap.String("encoding_format", string(r.EncodingFormat)),
			zap.String("user", r.User),
		}

		if !private {
			fields = append(fields, zap.Any("input", r.Input))
		}

		log.Info("openai embedding request", fields...)
	}
}

func logRequest(log *zap.Logger, prod, private bool, id string, r *goopenai.ChatCompletionRequest) {
	if prod {
		log.Info("openai chat completion request",
			zap.Time("createdAt", time.Now()),
			zap.String(correlationId, id),
			zap.Object("request", zapcore.ObjectMarshalerFunc(
				func(enc zapcore.ObjectEncoder) error {
					enc.AddString("model", r.Model)

					if len(r.Messages) != 0 {
						enc.AddArray("messages", zapcore.ArrayMarshalerFunc(
							func(enc zapcore.ArrayEncoder) error {
								for _, m := range r.Messages {
									err := enc.AppendObject(zapcore.ObjectMarshalerFunc(
										func(enc zapcore.ObjectEncoder) error {
											enc.AddString("name", m.Name)
											enc.AddString("role", m.Role)

											if m.FunctionCall != nil {
												enc.AddObject("function_call", zapcore.ObjectMarshalerFunc(
													func(enc zapcore.ObjectEncoder) error {
														enc.AddString("name", m.FunctionCall.Name)
														if !private {
															enc.AddString("arguments", m.FunctionCall.Arguments)
														}
														return nil
													},
												))
											}

											if !private {
												enc.AddString("content", m.Content)
											}

											return nil
										},
									))

									if err != nil {
										return err
									}
								}
								return nil
							},
						))
					}

					if len(r.Functions) != 0 {
						enc.AddArray("functions", zapcore.ArrayMarshalerFunc(
							func(enc zapcore.ArrayEncoder) error {
								for _, f := range r.Functions {
									err := enc.AppendObject(zapcore.ObjectMarshalerFunc(
										func(enc zapcore.ObjectEncoder) error {
											enc.AddString("name", f.Name)
											enc.AddString("description", f.Description)

											if f.Parameters != nil && !private {
												bs, err := json.Marshal(f.Parameters)
												if err != nil {
													return err
												}

												enc.AddString("parameters", string(bs))
											}

											return nil
										},
									))

									if err != nil {
										return err
									}

								}
								return nil
							},
						))
					}

					if r.MaxTokens != 0 {
						enc.AddInt("max_tokens", r.MaxTokens)
					}

					if r.Temperature != 0 {
						enc.AddFloat32("temperature", r.Temperature)
					}

					if r.TopP != 0 {
						enc.AddFloat32("top_p", r.TopP)
					}

					if r.N != 0 {
						enc.AddInt("n", r.N)
					}

					if r.Stream {
						enc.AddBool("stream", r.Stream)
					}

					if len(r.Stop) != 0 {
						enc.AddArray("stop", zapcore.ArrayMarshalerFunc(
							func(enc zapcore.ArrayEncoder) error {
								for _, s := range r.Stop {
									enc.AppendString(s)
								}
								return nil
							},
						))
					}

					if r.PresencePenalty != 0 {
						enc.AddFloat32("presence_penalty", r.PresencePenalty)
					}

					if r.FrequencyPenalty != 0 {
						enc.AddFloat32("frequency_penalty", r.FrequencyPenalty)
					}

					if len(r.LogitBias) != 0 {
						enc.AddObject("logit_bias", zapcore.ObjectMarshalerFunc(
							func(enc zapcore.ObjectEncoder) error {
								for k, v := range r.LogitBias {
									enc.AddInt(k, v)
								}
								return nil
							},
						))
					}

					if len(r.User) != 0 {
						enc.AddString("user", r.User)
					}

					return nil
				},
			)))
	}
}

func logOpenAiError(log *zap.Logger, prod bool, id string, errRes *goopenai.ErrorResponse) {
	if prod {
		log.Info("openai error response", zap.String(correlationId, id), zap.Any("error", errRes))
		return
	}

	log.Sugar().Infof("correlationId:%s | %s ", id, "openai error response")
}

func logError(log *zap.Logger, msg string, prod bool, id string, err error) {
	if prod {
		log.Debug(msg, zap.String(correlationId, id), zap.Error(err))
		return
	}

	log.Sugar().Debugf("correlationId:%s | %s | %v", id, msg, err)
}

func (ps *ProxyServer) Shutdown(ctx context.Context) error {
	if err := ps.server.Shutdown(ctx); err != nil {
		ps.log.Sugar().Infof("error shutting down proxy server: %v", err)

		return err
	}

	return nil
}
