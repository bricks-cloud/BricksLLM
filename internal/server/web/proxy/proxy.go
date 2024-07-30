package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/pii"
	"github.com/bricks-cloud/bricksllm/internal/policy"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ProviderSettingsManager interface {
	CreateSetting(setting *provider.Setting) (*provider.Setting, error)
	UpdateSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error)
	GetSettingViaCache(id string) (*provider.Setting, error)
}

type PoliciesManager interface {
	GetPolicyByIdFromMemdb(id string) *policy.Policy
}

type ProxyServer struct {
	server *http.Server
	log    *zap.Logger
}

type recorder interface {
	RecordEvent(e *event.Event) error
}

type KeyManager interface {
	GetKeys(tags, keyIds []string, provider string) ([]*key.ResponseKey, error)
	UpdateKey(id string, key *key.UpdateKey) (*key.ResponseKey, error)
	CreateKey(key *key.RequestKey) (*key.ResponseKey, error)
	DeleteKey(id string) error
}

type CustomProvidersManager interface {
	GetRouteConfigFromMem(name, path string) *custom.RouteConfig
	GetCustomProviderFromMem(name string) *custom.Provider
}

type Scanner interface {
	Scan(input []string) (*pii.Result, error)
}

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		a_or_b := func(a, b string) string {
			if a != "" {
				return a
			} else {
				return b
			}
		}
		c.Header("Access-Control-Allow-Origin", a_or_b(c.GetHeader("Origin"), "*"))
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", a_or_b(c.GetHeader("Access-Control-Request-Method"), "*"))
			c.Header("Access-Control-Allow-Headers", a_or_b(c.GetHeader("Access-Control-Request-Headers"), "*"))
			c.Header("Access-Control-Max-Age", "3600")
			c.AbortWithStatus(204)
		}
	}
}

func NewProxyServer(log *zap.Logger, mode, privacyMode string, c cache, m KeyManager, rm routeManager, a authenticator, psm ProviderSettingsManager, cpm CustomProvidersManager, ks keyStorage, e estimator, ae anthropicEstimator, aoe azureEstimator, v validator, r recorder, pub publisher, rlm rateLimitManager, timeOut time.Duration, ac accessCache, uac userAccessCache, pm PoliciesManager, scanner Scanner, cd CustomPolicyDetector, die deepinfraEstimator, um userManager, removeAgentHeaders bool) (*ProxyServer, error) {
	router := gin.New()
	prod := mode == "production"
	private := privacyMode == "strict"

	router.Use(CorsMiddleware())
	router.Use(getMiddleware(cpm, rm, pm, a, prod, private, log, pub, "proxy", ac, uac, http.Client{}, scanner, cd, um, removeAgentHeaders))

	client := http.Client{}

	// health check
	router.POST("/api/health", getGetHealthCheckHandler())

	// health check
	router.GET("/api/health", getGetHealthCheckHandler())

	// audios
	router.POST("/api/providers/openai/v1/audio/speech", getSpeechHandler(prod, client, timeOut))
	router.POST("/api/providers/openai/v1/audio/transcriptions", getTranscriptionsHandler(prod, client, timeOut, e))
	router.POST("/api/providers/openai/v1/audio/translations", getTranslationsHandler(prod, client, timeOut, e))

	// completions
	router.POST("/api/providers/openai/v1/chat/completions", getChatCompletionHandler(prod, private, client, e, timeOut))

	// embeddings
	router.POST("/api/providers/openai/v1/embeddings", getEmbeddingHandler(prod, private, client, e, timeOut))

	// moderations
	router.POST("/api/providers/openai/v1/moderations", getPassThroughHandler(prod, private, client, timeOut))

	// models
	router.GET("/api/providers/openai/v1/models", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/models/:model", getPassThroughHandler(prod, private, client, timeOut))
	router.DELETE("/api/providers/openai/v1/models/:model", getPassThroughHandler(prod, private, client, timeOut))

	// assistants
	router.POST("/api/providers/openai/v1/assistants", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/assistants/:assistant_id", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/assistants/:assistant_id", getPassThroughHandler(prod, private, client, timeOut))
	router.DELETE("/api/providers/openai/v1/assistants/:assistant_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/assistants", getPassThroughHandler(prod, private, client, timeOut))

	// assistant files
	router.POST("/api/providers/openai/v1/assistants/:assistant_id/files", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/assistants/:assistant_id/files/:file_id", getPassThroughHandler(prod, private, client, timeOut))
	router.DELETE("/api/providers/openai/v1/assistants/:assistant_id/files/:file_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/assistants/:assistant_id/files", getPassThroughHandler(prod, private, client, timeOut))

	// threads
	router.POST("/api/providers/openai/v1/threads", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id", getPassThroughHandler(prod, private, client, timeOut))
	router.DELETE("/api/providers/openai/v1/threads/:thread_id", getPassThroughHandler(prod, private, client, timeOut))

	// messages
	router.POST("/api/providers/openai/v1/threads/:thread_id/messages", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/messages/:message_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages", getPassThroughHandler(prod, private, client, timeOut))

	// message files
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files", getPassThroughHandler(prod, private, client, timeOut))

	// runs
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs/:run_id", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs/:run_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/threads/runs", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps", getPassThroughHandler(prod, private, client, timeOut))

	// files
	router.GET("/api/providers/openai/v1/files", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/files", getPassThroughHandler(prod, private, client, timeOut))
	router.DELETE("/api/providers/openai/v1/files/:file_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/files/:file_id", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/files/:file_id/content", getPassThroughHandler(prod, private, client, timeOut))

	// batch
	router.POST("/api/providers/openai/v1/batches", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/batches/:batch_id", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/batches/:batch_id/cancel", getPassThroughHandler(prod, private, client, timeOut))
	router.GET("/api/providers/openai/v1/batches", getPassThroughHandler(prod, private, client, timeOut))

	// images
	router.POST("/api/providers/openai/v1/images/generations", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/images/edits", getPassThroughHandler(prod, private, client, timeOut))
	router.POST("/api/providers/openai/v1/images/variations", getPassThroughHandler(prod, private, client, timeOut))

	// azure
	router.POST("/api/providers/azure/openai/deployments/:deployment_id/chat/completions", getAzureChatCompletionHandler(prod, private, client, aoe, timeOut))
	router.POST("/api/providers/azure/openai/deployments/:deployment_id/embeddings", getAzureEmbeddingsHandler(prod, private, client, aoe, timeOut))
	router.POST("/api/providers/azure/openai/deployments/:deployment_id/completions", getAzureCompletionsHandler(prod, private, client, aoe, timeOut))

	// anthropic
	router.POST("/api/providers/anthropic/v1/complete", getCompletionHandler(prod, private, client, timeOut))
	router.POST("/api/providers/anthropic/v1/messages", getMessagesHandler(prod, private, client, ae, timeOut))

	// vllm
	router.POST("/api/providers/vllm/v1/chat/completions", getVllmChatCompletionsHandler(prod, private, client, timeOut))
	router.POST("/api/providers/vllm/v1/completions", getVllmCompletionsHandler(prod, private, client, timeOut))

	// deepinfra
	router.POST("/api/providers/deepinfra/v1/chat/completions", getDeepinfraChatCompletionsHandler(prod, private, client, timeOut))
	router.POST("/api/providers/deepinfra/v1/completions", getDeepinfraCompletionsHandler(prod, private, client, timeOut))
	router.POST("/api/providers/deepinfra/v1/embeddings", getDeepinfraEmbeddingsHandler(prod, private, client, die, timeOut))

	// custom provider
	router.POST("/api/custom/providers/:provider/*wildcard", getCustomProviderHandler(prod, client, timeOut))

	// custom route
	router.POST("/api/routes/*route", getRouteHandler(prod, c, aoe, e, client, r))

	// vector store
	router.POST("/api/providers/openai/v1/vector_stores", getCreateVectorStoreHandler(prod, client, timeOut))
	router.GET("/api/providers/openai/v1/vector_stores", getListVectorStoresHandler(prod, client, timeOut))
	router.GET("/api/providers/openai/v1/vector_stores/:vector_store_id", getGetVectorStoreHandler(prod, client, timeOut))
	router.POST("/api/providers/openai/v1/vector_stores/:vector_store_id", getModifyVectorStoreHandler(prod, client, timeOut))
	router.DELETE("/api/providers/openai/v1/vector_stores/:vector_store_id", getDeleteVectorStoreHandler(prod, client, timeOut))

	// vector store files
	router.POST("/api/providers/openai/v1/vector_stores/:vector_store_id/files", getCreateVectorStoreFileHandler(prod, client, timeOut))
	router.GET("/api/providers/openai/v1/vector_stores/:vector_store_id/files", getListVectorStoreFilesHandler(prod, client, timeOut))
	router.GET("/api/providers/openai/v1/vector_stores/:vector_store_id/files/:file_id", getGetVectorStoreFileHandler(prod, client, timeOut))
	router.DELETE("/api/providers/openai/v1/vector_stores/:vector_store_id/files/:file_id", getDeleteVectorStoreFileHandler(prod, client, timeOut))

	// vector store file batches
	router.POST("/api/providers/openai/v1/vector_stores/:vector_store_id/file_batches", getCreateVectorStoreFileBatchHandler(prod, client, timeOut))
	router.GET("/api/providers/openai/v1/vector_stores/:vector_store_id/file_batches/:batch_id", getGetVectorStoreFileBatchHandler(prod, client, timeOut))
	router.POST("/api/providers/openai/v1/vector_stores/:vector_store_id/file_batches/:batch_id/cancel", getCancelVectorStoreFileBatchHandler(prod, client, timeOut))
	router.GET("/api/providers/openai/v1/vector_stores/:vector_store_id/file_batches/:batch_id/files", getListVectorStoreFileBatchFilesHandler(prod, client, timeOut))

	srv := &http.Server{
		Addr:    ":8002",
		Handler: router,
	}

	return &ProxyServer{
		log:    log,
		server: srv,
	}, nil
}

func getGetHealthCheckHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
}

type Form struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type ImageEditForm struct {
	Image *multipart.FileHeader `form:"image" binding:"required"`
	Mask  *multipart.FileHeader `form:"mask" binding:"required"`
}

type ImageVariationForm struct {
	Image *multipart.FileHeader `form:"image" binding:"required"`
}

type TransriptionForm struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type TranslationForm struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

func writeFieldToBuffer(fields []string, c *gin.Context, writer *multipart.Writer, overWrites map[string]string) error {
	for _, field := range fields {
		val := c.PostForm(field)

		if len(overWrites) != 0 {
			if ow := overWrites[field]; len(ow) != 0 {
				val = ow
			}
		}

		if len(val) != 0 {
			err := writer.WriteField(field, val)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getPassThroughHandler(prod, private bool, client http.Client, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)

		tags := []string{
			fmt.Sprintf("path:%s", c.FullPath()),
		}

		telemetry.Incr("bricksllm.proxy.get_pass_through_handler.requests", tags, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		targetUrl, err := buildProxyUrl(c)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_pass_through_handler.proxy_url_not_found", tags, 1)
			logError(log, "error when building proxy url", prod, err)
			JSON(c, http.StatusNotFound, "[BricksLLM] cannot find corresponding proxy url")
			return
		}

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, targetUrl, c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		// copy query params
		req.URL.RawQuery = c.Request.URL.RawQuery

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodPost {
			purpose := c.PostForm("purpose")

			var b bytes.Buffer
			writer := multipart.NewWriter(&b)
			err := writer.WriteField("purpose", purpose)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_pass_through_handler.write_field_error", tags, 1)
				logError(log, "error when writing field", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field")
				return
			}

			var form Form
			c.ShouldBind(&form)

			fieldWriter, err := writer.CreateFormFile("file", form.File.Filename)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_pass_through_handler.create_form_file_error", tags, 1)
				logError(log, "error when creating form file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create form file")
				return
			}

			opened, err := form.File.Open()
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_pass_through_handler.open_file_error", tags, 1)
				logError(log, "error when openning file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open file")
				return
			}

			_, err = io.Copy(fieldWriter, opened)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_pass_through_handler.copy_file_error", tags, 1)
				logError(log, "error when copying file", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy file")
				return
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			writer.Close()

			req.Body = io.NopCloser(&b)
		}

		if c.FullPath() == "/api/providers/openai/v1/images/edits" && c.Request.Method == http.MethodPost {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			err := writeFieldToBuffer([]string{
				"prompt",
				"model",
				"n",
				"size",
				"response_format",
				"user",
			}, c, writer, nil)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", tags, 1)
				logError(log, "error when writing field to buffer", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
				return
			}

			var form ImageEditForm
			c.ShouldBind(&form)

			if form.Image != nil {
				fieldWriter, err := writer.CreateFormFile("image", form.Image.Filename)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.create_image_file_error", tags, 1)
					logError(log, "error when creating form file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create image file")
					return
				}

				opened, err := form.Image.Open()
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.open_image_file_error", tags, 1)
					logError(log, "error when openning file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open image file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.copy_image_file_error", tags, 1)
					logError(log, "error when copying image file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy image file")
					return
				}
			}

			if form.Mask != nil {
				fieldWriter, err := writer.CreateFormFile("mask", form.Mask.Filename)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.create_mask_file_error", tags, 1)
					logError(log, "error when creating form file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create mask file")
					return
				}

				opened, err := form.Image.Open()
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.open_mask_file_error", tags, 1)
					logError(log, "error when openning file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open mask file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.copy_mask_file_error", tags, 1)
					logError(log, "error when copying mask file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy mask file")
					return
				}
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			writer.Close()

			req.Body = io.NopCloser(&b)
		}

		if c.FullPath() == "/api/providers/openai/v1/images/variations" && c.Request.Method == http.MethodPost {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			err := writeFieldToBuffer([]string{
				"model",
				"n",
				"size",
				"response_format",
				"user",
			}, c, writer, nil)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", tags, 1)
				logError(log, "error when writing field to buffer", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
				return
			}

			var form ImageVariationForm
			c.ShouldBind(&form)

			if form.Image != nil {
				fieldWriter, err := writer.CreateFormFile("image", form.Image.Filename)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.create_image_file_error", tags, 1)
					logError(log, "error when creating form file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create image file")
					return
				}

				opened, err := form.Image.Open()
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.open_image_file_error", tags, 1)
					logError(log, "error when openning file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open image file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_pass_through_handler.copy_image_file_error", tags, 1)
					logError(log, "error when copying file", prod, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy image file")
					return
				}
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			writer.Close()

			req.Body = io.NopCloser(&b)
		}

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_pass_through_handler.http_client_error", tags, 1)

			logError(log, "error when sending pass through request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send pass through request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_pass_through_handler.latency", dur, tags, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai pass through response body")
			return
		}

		if res.StatusCode == http.StatusOK {
			telemetry.Incr("bricksllm.proxy.get_pass_through_handler.success", tags, 1)
			telemetry.Timing("bricksllm.proxy.get_pass_through_handler.success_latency", dur, tags, 1)

			if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodPost {
				logAssistantResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodGet {
				logAssistantResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodPost {
				logAssistantResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodDelete {
				logDeleteAssistantResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodGet {
				logListAssistantsResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodPost {
				logAssistantFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodGet {
				logAssistantFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodDelete {
				logDeleteAssistantFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodGet {
				logListAssistantFilesResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads" && c.Request.Method == http.MethodPost {
				logThreadResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodGet {
				logThreadResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodPost {
				logThreadResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodDelete {
				logDeleteThreadResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodPost {
				logMessageResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodGet {
				logMessageResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodPost {
				logMessageResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodGet {
				logListMessagesResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id" && c.Request.Method == http.MethodGet {
				logRetrieveMessageFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files" && c.Request.Method == http.MethodGet {
				logListMessageFilesResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodGet {
				logRunResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodGet {
				logListRunsResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/runs" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id" && c.Request.Method == http.MethodGet {
				logRetrieveRunStepResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps" && c.Request.Method == http.MethodGet {
				logListRunStepsResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/moderations" && c.Request.Method == http.MethodPost {
				logCreateModerationResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/models" && c.Request.Method == http.MethodGet {
				logListModelsResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodGet {
				logRetrieveModelResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodDelete {
				logDeleteModelResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodGet {
				logListFilesResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodPost {
				logUploadFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodDelete {
				logDeleteFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodGet {
				logRetrieveFileResponse(log, bytes, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/files/:file_id/content" && c.Request.Method == http.MethodGet {
				logRetrieveFileContentResponse(log, prod)
			}

			if c.FullPath() == "/api/providers/openai/v1/images/generations" && c.Request.Method == http.MethodPost {
				logImageResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/images/edits" && c.Request.Method == http.MethodPost {
				logImageResponse(log, bytes, prod, private)
			}

			if c.FullPath() == "/api/providers/openai/v1/images/variations" && c.Request.Method == http.MethodPost {
				logImageResponse(log, bytes, prod, private)
			}
		}

		if res.StatusCode != http.StatusOK {
			telemetry.Timing("bricksllm.proxy.get_pass_through_handler.error_latency", dur, tags, 1)
			telemetry.Incr("bricksllm.proxy.get_pass_through_handler.error_response", tags, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai pass through error response body", prod, err)
			}

			logOpenAiError(log, prod, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if len(res.Header.Get("content-type")) != 0 {
			c.Data(res.StatusCode, res.Header.Get("content-type"), bytes)
			return
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func buildProxyUrl(c *gin.Context) (string, error) {
	if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/assistants", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodDelete {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/assistants", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id") + "/files", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id") + "/files/" + c.Param("file_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodDelete {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id") + "/files/" + c.Param("file_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/assistants/" + c.Param("assistant_id") + "/files", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodDelete {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/messages", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/messages/" + c.Param("message_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/messages/" + c.Param("message_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/messages", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/messages/" + c.Param("message_id") + "/files/" + c.Param("file_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/messages/" + c.Param("message_id") + "/files", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs/" + c.Param("run_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs/" + c.Param("run_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs/" + c.Param("run_id") + "/submit_tool_outputs", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs/" + c.Param("run_id") + "/cancel", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/runs" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/threads/runs", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs/" + c.Param("run_id") + "/steps/" + c.Param("step_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/threads/" + c.Param("thread_id") + "/runs/" + c.Param("run_id") + "/steps", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/moderations" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/moderations", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/models" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/models", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/models/" + c.Param("model"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodDelete {
		return "https://api.openai.com/v1/models/" + c.Param("model"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/files", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/files", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodDelete {
		return "https://api.openai.com/v1/files/" + c.Param("file_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/files/" + c.Param("file_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/files/:file_id/content" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/files/" + c.Param("file_id") + "/content", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/batches" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/batches", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/batches/:batch_id" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/batches/" + c.Param("batch_id"), nil
	}

	if c.FullPath() == "/api/providers/openai/v1/batches/:batch_id/cancel" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/batches/" + c.Param("batch_id") + "/cancel", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/batches" && c.Request.Method == http.MethodGet {
		return "https://api.openai.com/v1/batches", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/images/generations" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/images/generations", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/images/edits" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/images/edits", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/images/variations" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/images/variations", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/audio/speech" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/audio/speech", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/audio/transcriptions" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/audio/transcriptions", nil
	}

	if c.FullPath() == "/api/providers/openai/v1/audio/translations" && c.Request.Method == http.MethodPost {
		return "https://api.openai.com/v1/audio/translations", nil
	}

	return "", errors.New("cannot find corresponding OpenAI target proxy")
}

var (
	headerData            = []byte("data: ")
	eventCompletionPrefix = []byte("event: completion")
	eventPingPrefix       = []byte("event: ping")
	eventErrorPrefix      = []byte("event: error")
)

func (ps *ProxyServer) Run() {
	go func() {
		ps.log.Info("proxy server listening at 8002")

		// health check
		ps.log.Info("PORT 8002 | GET    | /api/health is ready")

		// audio
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/audio/speech is ready for creating openai speeches")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/audio/transcriptions is ready for creating openai transcriptions")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/audio/translations is ready for creating openai translations")

		// chat completions
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/chat/completions is ready for forwarding chat completion requests to openai")

		// embeddings
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/embeddings is ready for forwarding embeddings requests to openai")

		// moderations
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/moderations is ready for forwarding moderation requests to openai")

		// models
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/models is ready for listing openai models")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/models/:model is ready for retrieving an openai model")

		// files
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/files is ready for listing files from openai")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/files is ready for uploading files to openai")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/files/:file_id is ready for retrieving a file metadata from openai")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/files/:file_id/content is ready for retrieving a file's content from openai")

		// assistants
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/assistants is ready for creating openai assistants")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/assistants/:assistant_id is ready for retrieving an openai assistant")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/assistants/:assistant_id is ready for modifying an openai assistant")
		ps.log.Info("PORT 8002 | DELETE | /api/providers/openai/v1/assistants/:assistant_id is ready for deleting an openai assistant")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/assistants is ready for retrieving openai assistants")

		// assistant files
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/assistants/:assistant_id/files is ready for creating openai assistant file")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/assistants/:assistant_id/files/:file_id is ready for retrieving openai assistant file")
		ps.log.Info("PORT 8002 | DELETE | /api/providers/openai/v1/assistants/:assistant_id/files/:file_id is ready for deleting openai assistant file")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/assistants/:assistant_id/files is ready for retireving openai assistant files")

		// threads
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads is ready for creating an openai thread")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id is ready for retrieving an openai thread")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id is ready for modifying an openai thread")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id is ready for deleting an openai thread")

		// messages
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/messages is ready for creating an openai message")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/messages/:message_id is ready for retrieving an openai message")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/messages/:message_id is ready for modifying an openai message")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/messages is ready for retrieving openai messages")

		// message files
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id is ready for retrieving an openai message file")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/messages/:message_id/files is ready for retrieving openai message files")

		// runs
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/runs is ready for creating an openai run")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/runs/:run_id is ready for retrieving an openai run")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/runs/:run_id is ready for modifying an openai run")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/runs is ready for retrieving openai runs")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs is ready for submitting tool outputs to an openai run")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel is ready for cancelling an openai run")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/runs is ready for creating an openai thread and run")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id is ready for retrieving an openai run step")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps is ready for retrieving openai run steps")

		// images
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/images/generations is ready for generating openai images")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/images/edits is ready for editting openi images")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/images/variations is ready for generating openai image variations")

		// azure
		ps.log.Info("PORT 8002 | POST   | /api/providers/azure/openai/deployments/:deployment_id/chat/completions is ready for forwarding completion requests to azure openai")
		ps.log.Info("PORT 8002 | POST   | /api/providers/azure/openai/deployments/:deployment_id/embeddings is ready for forwarding embeddings requests to azure openai")

		// anthropic
		ps.log.Info("PORT 8002 | POST   | /api/providers/anthropic/v1/complete is ready for forwarding completion requests to anthropic")
		ps.log.Info("PORT 8002 | POST   | /api/providers/anthropic/v1/messages is ready for forwarding message requests to anthropic")

		// vllm
		ps.log.Info("PORT 8002 | POST   | /api/providers/vllm/v1/chat/completions is ready for forwarding vllm chat completions requests")
		ps.log.Info("PORT 8002 | POST   | /api/providers/vllm/v1/completions is ready for forwarding vllm completions requests")

		// deepinfra
		ps.log.Info("PORT 8002 | POST   | /api/providers/deepinfra/v1/chat/completions is ready for forwarding deepinfra chat completions requests")
		ps.log.Info("PORT 8002 | POST   | /api/providers/deepinfra/v1/completions is ready for forwarding deepinfra completions requests")
		ps.log.Info("PORT 8002 | POST   | /api/providers/deepinfra/v1/embeddings is ready for forwarding deepinfra embeddings requests")

		// custom provider
		ps.log.Info("PORT 8002 | POST   | /api/custom/providers/:provider/*wildcard is ready for forwarding requests to custom providers")

		// custom route
		ps.log.Info("PORT 8002 | POST   | /api/routes/*route is ready for forwarding requests to a custom route")

		// vector store
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/vector_stores is ready for creating an openai vector store")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/vector_stores is ready for listing openai vector stores")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/vector_stores/:vector_store_id is ready for getting an openai vector store")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/vector_stores/:vector_store_id is ready for modifying an openai vector store")
		ps.log.Info("PORT 8002 | DELETE | /api/providers/openai/v1/vector_stores/:vector_store_id is ready for deleting an openai vector store")

		// vector store files
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/vector_stores/:vector_store_id/files is ready for creating an openai vector store file")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/vector_stores/:vector_store_id/files is ready for listing openai vector store files")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/vector_stores/:vector_store_id/files/:file_id is ready for getting an openai vector store file")
		ps.log.Info("PORT 8002 | DELETE | /api/providers/openai/v1/vector_stores/:vector_store_id/files/:file_id is ready for deleting an openai vector store file")

		// vector store file batches
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/vector_stores/:vector_store_id/file_batches is ready for creating an openai vector store file batch")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/vector_stores/:vector_store_id/file_batches/:batch_id is ready for getting an openai vector store file batch")
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/vector_stores/:vector_store_id/file_batches/:batch_id/cancel is ready for cancelling an openai vector store file batch")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/vector_stores/:vector_store_id/file_batches/:batch_id/files is ready for listing openai vector store file batch files")

		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ps.log.Sugar().Fatalf("error proxy server listening: %v", err)
			return
		}
	}()
}

func logEmbeddingResponse(log *zap.Logger, prod, private bool, r *EmbeddingResponse) {
	if prod {
		log.Info("openai embeddings response",
			zap.Time("createdAt", time.Now()),
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

func logBase64EmbeddingResponse(log *zap.Logger, prod, private bool, r *EmbeddingResponseBase64) {
	if prod {
		log.Info("openai embeddings response",
			zap.Time("createdAt", time.Now()),
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

func logChatCompletionResponse(log *zap.Logger, prod, private bool, r *goopenai.ChatCompletionResponse) {
	if prod {
		log.Info("openai chat completion response",
			zap.Time("createdAt", time.Now()),
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

func logEmbeddingRequest(log *zap.Logger, prod, private bool, r *goopenai.EmbeddingRequest) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", string(r.Model)),
			zap.String("encoding_format", string(r.EncodingFormat)),
			zap.String("user", r.User),
		}

		if !private {
			fields = append(fields, zap.Any("input", r.Input))
		}

		log.Info("openai embedding request", fields...)
	}
}

func logRequest(log *zap.Logger, prod, private bool, r *goopenai.ChatCompletionRequest) {
	if prod {
		log.Info("openai chat completion request",
			zap.Time("createdAt", time.Now()),
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

func logOpenAiError(log *zap.Logger, prod bool, errRes *goopenai.ErrorResponse) {
	if prod {
		log.Info("openai error response", zap.Any("error", errRes))
		return
	}

	log.Info("openai error response")
}

func logError(log *zap.Logger, msg string, prod bool, err error) {
	if prod {
		log.Debug(msg, zap.Error(err))
		return
	}
	log.Debug(fmt.Sprintf("%s | %v", msg, err))
}

func (ps *ProxyServer) Shutdown(ctx context.Context) error {
	if err := ps.server.Shutdown(ctx); err != nil {
		ps.log.Sugar().Infof("error shutting down proxy server: %v", err)

		return err
	}

	return nil
}
