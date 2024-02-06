package proxy

import (
	"bufio"
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
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ProviderSettingsManager interface {
	CreateSetting(setting *provider.Setting) (*provider.Setting, error)
	UpdateSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error)
	GetSetting(id string) (*provider.Setting, error)
}

const (
	correlationId string = "correlationId"
)

type ProxyServer struct {
	server *http.Server
	log    *zap.Logger
}

type recorder interface {
	// RecordKeySpend(keyId string, micros int64, costLimitUnit key.TimeUnit) error
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

func NewProxyServer(log *zap.Logger, mode, privacyMode string, c cache, m KeyManager, rm routeManager, a authenticator, psm ProviderSettingsManager, cpm CustomProvidersManager, ks keyStorage, kms keyMemStorage, e estimator, ae anthropicEstimator, aoe azureEstimator, v validator, r recorder, pub publisher, rlm rateLimitManager, timeOut time.Duration, ac accessCache) (*ProxyServer, error) {
	router := gin.New()
	prod := mode == "production"
	private := privacyMode == "strict"

	router.Use(getMiddleware(kms, cpm, rm, a, prod, private, e, ae, aoe, v, ks, log, rlm, pub, "proxy", ac))

	client := http.Client{}

	// health check
	router.POST("/api/health", getGetHealthCheckHandler())

	// audios
	router.POST("/api/providers/openai/v1/audio/speech", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/audio/transcriptions", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/audio/translations", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// completions
	router.POST("/api/providers/openai/v1/chat/completions", getChatCompletionHandler(r, prod, private, psm, client, kms, log, e, timeOut))

	// embeddings
	router.POST("/api/providers/openai/v1/embeddings", getEmbeddingHandler(r, prod, private, psm, client, kms, log, e, timeOut))

	// moderations
	router.POST("/api/providers/openai/v1/moderations", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// models
	router.GET("/api/providers/openai/v1/models", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/models/:model", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.DELETE("/api/providers/openai/v1/models/:model", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// assistants
	router.POST("/api/providers/openai/v1/assistants", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/assistants/:assistant_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/assistants/:assistant_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.DELETE("/api/providers/openai/v1/assistants/:assistant_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/assistants", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// assistant files
	router.POST("/api/providers/openai/v1/assistants/:assistant_id/files", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/assistants/:assistant_id/files/:file_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.DELETE("/api/providers/openai/v1/assistants/:assistant_id/files/:file_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/assistants/:assistant_id/files", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// threads
	router.POST("/api/providers/openai/v1/threads", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.DELETE("/api/providers/openai/v1/threads/:thread_id", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// messages
	router.POST("/api/providers/openai/v1/threads/:thread_id/messages", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/messages/:message_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// message files
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// runs
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs/:run_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs/:run_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/threads/runs", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// files
	router.GET("/api/providers/openai/v1/files", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/files", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.DELETE("/api/providers/openai/v1/files/:file_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/files/:file_id", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.GET("/api/providers/openai/v1/files/:file_id/content", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// images
	router.POST("/api/providers/openai/v1/images/generations", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/images/edits", getPassThroughHandler(r, prod, private, client, log, timeOut))
	router.POST("/api/providers/openai/v1/images/variations", getPassThroughHandler(r, prod, private, client, log, timeOut))

	// azure
	router.POST("/api/providers/azure/openai/deployments/:deployment_id/chat/completions", getAzureChatCompletionHandler(r, prod, private, psm, client, kms, log, aoe, timeOut))
	router.POST("/api/providers/azure/openai/deployments/:deployment_id/embeddings", getAzureEmbeddingsHandler(r, prod, private, psm, client, kms, log, aoe, timeOut))

	// anthropic
	router.POST("/api/providers/anthropic/v1/complete", getCompletionHandler(r, prod, private, client, kms, log, ae, timeOut))

	// custom provider
	router.POST("/api/custom/providers/:provider/*wildcard", getCustomProviderHandler(prod, private, psm, cpm, client, log, timeOut))

	// custom route
	router.POST("/api/routes/*route", getRouteHandler(prod, private, rm, c, aoe, e, r, client, log, timeOut))

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

func writeFieldToBuffer(fields []string, c *gin.Context, writer *multipart.Writer) error {
	for _, field := range fields {
		val := c.PostForm(field)
		if len(val) != 0 {
			err := writer.WriteField(field, val)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getPassThroughHandler(r recorder, prod, private bool, client http.Client, log *zap.Logger, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		tags := []string{
			fmt.Sprintf("path:%s", c.FullPath()),
		}

		stats.Incr("bricksllm.proxy.get_pass_through_handler.requests", tags, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(correlationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		targetUrl, err := buildProxyUrl(c)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_pass_through_handler.proxy_url_not_found", tags, 1)
			logError(log, "error when building proxy url", prod, cid, err)
			JSON(c, http.StatusNotFound, "[BricksLLM] cannot find corresponding proxy url")
			return
		}

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, targetUrl, c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodPost {
			purpose := c.PostForm("purpose")

			var b bytes.Buffer
			writer := multipart.NewWriter(&b)
			err := writer.WriteField("purpose", purpose)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.write_field_error", tags, 1)
				logError(log, "error when writing field", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field")
				return
			}

			var form Form
			c.ShouldBind(&form)

			fieldWriter, err := writer.CreateFormFile("file", form.File.Filename)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.create_form_file_error", tags, 1)
				logError(log, "error when creating form file", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create form file")
				return
			}

			opened, err := form.File.Open()
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.open_file_error", tags, 1)
				logError(log, "error when openning file", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open file")
				return
			}

			_, err = io.Copy(fieldWriter, opened)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_file_error", tags, 1)
				logError(log, "error when copying file", prod, cid, err)
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
			}, c, writer)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", tags, 1)
				logError(log, "error when writing field to buffer", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
				return
			}

			var form ImageEditForm
			c.ShouldBind(&form)

			if form.Image != nil {
				fieldWriter, err := writer.CreateFormFile("image", form.Image.Filename)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.create_image_file_error", tags, 1)
					logError(log, "error when creating form file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create image file")
					return
				}

				opened, err := form.Image.Open()
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.open_image_file_error", tags, 1)
					logError(log, "error when openning file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open image file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_image_file_error", tags, 1)
					logError(log, "error when copying image file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy image file")
					return
				}
			}

			if form.Mask != nil {
				fieldWriter, err := writer.CreateFormFile("mask", form.Mask.Filename)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.create_mask_file_error", tags, 1)
					logError(log, "error when creating form file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create mask file")
					return
				}

				opened, err := form.Image.Open()
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.open_mask_file_error", tags, 1)
					logError(log, "error when openning file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open mask file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_mask_file_error", tags, 1)
					logError(log, "error when copying mask file", prod, cid, err)
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
			}, c, writer)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", tags, 1)
				logError(log, "error when writing field to buffer", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
				return
			}

			var form ImageVariationForm
			c.ShouldBind(&form)

			if form.Image != nil {
				fieldWriter, err := writer.CreateFormFile("image", form.Image.Filename)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.create_image_file_error", tags, 1)
					logError(log, "error when creating form file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create image file")
					return
				}

				opened, err := form.Image.Open()
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.open_image_file_error", tags, 1)
					logError(log, "error when openning file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open image file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_image_file_error", tags, 1)
					logError(log, "error when copying file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy image file")
					return
				}
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			writer.Close()

			req.Body = io.NopCloser(&b)
		}

		if c.FullPath() == "/api/providers/openai/v1/audio/transcriptions" && c.Request.Method == http.MethodPost {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			err := writeFieldToBuffer([]string{
				"model",
				"language",
				"prompt",
				"response_format",
				"temperature",
			}, c, writer)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", tags, 1)
				logError(log, "error when writing field to buffer", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
				return
			}

			var form TransriptionForm
			c.ShouldBind(&form)

			if form.File != nil {
				fieldWriter, err := writer.CreateFormFile("file", form.File.Filename)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.create_transcription_file_error", tags, 1)
					logError(log, "error when creating transcription file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create transcription file")
					return
				}

				opened, err := form.File.Open()
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.open_transcription_file_error", tags, 1)
					logError(log, "error when openning transcription file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open transcription file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_transcription_file_error", tags, 1)
					logError(log, "error when copying transcription file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy transcription file")
					return
				}
			}

			req.Header.Set("Content-Type", writer.FormDataContentType())

			writer.Close()

			req.Body = io.NopCloser(&b)
		}

		if c.FullPath() == "/api/providers/openai/v1/audio/translations" && c.Request.Method == http.MethodPost {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			err := writeFieldToBuffer([]string{
				"model",
				"prompt",
				"response_format",
				"temperature",
			}, c, writer)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_pass_through_handler.write_field_to_buffer_error", tags, 1)
				logError(log, "error when writing field to buffer", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot write field to buffer")
				return
			}

			var form TranslationForm
			c.ShouldBind(&form)

			if form.File != nil {
				fieldWriter, err := writer.CreateFormFile("file", form.File.Filename)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.create_translation_file_error", tags, 1)
					logError(log, "error when creating translation file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot create translation file")
					return
				}

				opened, err := form.File.Open()
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.open_translation_file_error", tags, 1)
					logError(log, "error when openning translation file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot open translation file")
					return
				}

				_, err = io.Copy(fieldWriter, opened)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_pass_through_handler.copy_translation_file_error", tags, 1)
					logError(log, "error when copying translation file", prod, cid, err)
					JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot copy translation file")
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
			stats.Incr("bricksllm.proxy.get_pass_through_handler.http_client_error", tags, 1)

			logError(log, "error when sending embedding request to openai", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send embedding request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Now().Sub(start)
		stats.Timing("bricksllm.proxy.get_pass_through_handler.latency", dur, tags, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai embedding response body")
			return
		}

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_pass_through_handler.success", tags, 1)
			stats.Timing("bricksllm.proxy.get_pass_through_handler.success_latency", dur, tags, 1)

			if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodPost {
				logAssistantResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodGet {
				logAssistantResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodPost {
				logAssistantResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id" && c.Request.Method == http.MethodDelete {
				logDeleteAssistantResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants" && c.Request.Method == http.MethodGet {
				logListAssistantFilesResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodPost {
				logAssistantFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodGet {
				logAssistantFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files/:file_id" && c.Request.Method == http.MethodDelete {
				logDeleteAssistantFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/assistants/:assistant_id/files" && c.Request.Method == http.MethodGet {
				logListAssistantFilesResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads" && c.Request.Method == http.MethodPost {
				logThreadResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodGet {
				logThreadResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodPost {
				logThreadResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id" && c.Request.Method == http.MethodDelete {
				logDeleteThreadResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodPost {
				logMessageResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodGet {
				logMessageResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id" && c.Request.Method == http.MethodPost {
				logMessageResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages" && c.Request.Method == http.MethodGet {
				logListMessagesResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files/:file_id" && c.Request.Method == http.MethodGet {
				logRetrieveMessageFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/messages/:message_id/files" && c.Request.Method == http.MethodGet {
				logListMessageFilesResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodGet {
				logRunResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs" && c.Request.Method == http.MethodGet {
				logListRunsResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/submit_tool_outputs" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/cancel" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/runs" && c.Request.Method == http.MethodPost {
				logRunResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps/:step_id" && c.Request.Method == http.MethodGet {
				logRetrieveRunStepResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/threads/:thread_id/runs/:run_id/steps" && c.Request.Method == http.MethodGet {
				logListRunStepsResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/moderations" && c.Request.Method == http.MethodPost {
				logCreateModerationResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/models" && c.Request.Method == http.MethodGet {
				logListModelsResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodGet {
				logRetrieveModelResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/models/:model" && c.Request.Method == http.MethodDelete {
				logDeleteModelResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodGet {
				logListFilesResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/files" && c.Request.Method == http.MethodPost {
				logUploadFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodDelete {
				logDeleteFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/files/:file_id" && c.Request.Method == http.MethodGet {
				logRetrieveFileResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/files/:file_id/content" && c.Request.Method == http.MethodGet {
				logRetrieveFileContentResponse(log, bytes, prod, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/images/generations" && c.Request.Method == http.MethodPost {
				logImageResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/images/edits" && c.Request.Method == http.MethodPost {
				logImageResponse(log, bytes, prod, private, cid)
			}

			if c.FullPath() == "/api/providers/openai/v1/images/variations" && c.Request.Method == http.MethodPost {
				logImageResponse(log, bytes, prod, private, cid)
			}
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_pass_through_handler.error_latency", dur, tags, 1)
			stats.Incr("bricksllm.proxy.get_pass_through_handler.error_response", tags, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai pass through error response body", prod, cid, err)
			}

			logOpenAiError(log, prod, cid, errorRes)
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

func getEmbeddingHandler(r recorder, prod, private bool, psm ProviderSettingsManager, client http.Client, kms keyMemStorage, log *zap.Logger, e estimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.proxy.get_embedding_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		// raw, exists := c.Get("key")
		// kc, ok := raw.(*key.ResponseKey)
		// if !exists || !ok {
		// 	stats.Incr("bricksllm.proxy.get_embedding_handler.api_key_not_registered", nil, 1)
		// 	JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
		// 	return
		// }

		id := c.GetString(correlationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, "https://api.openai.com/v1/embeddings", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_embedding_handler.http_client_error", nil, 1)

			logError(log, "error when sending embedding request to openai", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send embedding request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Now().Sub(start)
		stats.Timing("bricksllm.proxy.get_embedding_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai embedding response body")
			return
		}

		var cost float64 = 0
		chatRes := &EmbeddingResponse{}
		promptTokenCounts := 0
		base64ChatRes := &EmbeddingResponseBase64{}
		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_embedding_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_embedding_handler.success_latency", dur, nil, 1)

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
					promptTokenCounts = base64ChatRes.Usage.PromptTokens
					totalTokens = base64ChatRes.Usage.TotalTokens
				}

				if format != "base64" {
					logEmbeddingResponse(log, prod, private, id, chatRes)
					promptTokenCounts = chatRes.Usage.PromptTokens
					totalTokens = chatRes.Usage.TotalTokens
				}

				cost, err = e.EstimateEmbeddingsInputCost(model, totalTokens)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_embedding_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost for embedding", prod, id, err)
				}

				// micros := int64(cost * 1000000)
				// err = r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
				// if err != nil {
				// 	stats.Incr("bricksllm.proxy.get_embedding_handler.record_key_spend_error", nil, 1)
				// 	logError(log, "error when recording openai spend for embedding", prod, id, err)
				// }
			}
		}

		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", promptTokenCounts)

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_embedding_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_embedding_handler.error_response", nil, 1)

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

var (
	headerData            = []byte("data: ")
	eventCompletionPrefix = []byte("event: completion")
	eventPingPrefix       = []byte("event: ping")
	eventErrorPrefix      = []byte("event: error")
	errorPrefix           = []byte(`data: {"error":`)
)

func getChatCompletionHandler(r recorder, prod, private bool, psm ProviderSettingsManager, client http.Client, kms keyMemStorage, log *zap.Logger, e estimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.proxy.get_chat_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(correlationId)
		// raw, exists := c.Get("key")
		// kc, ok := raw.(*key.ResponseKey)
		// if !exists || !ok {
		// 	stats.Incr("bricksllm.proxy.get_chat_completion_handler.api_key_not_registered", nil, 1)
		// 	JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
		// 	return
		// }

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		isStreaming := c.GetBool("stream")
		if isStreaming {
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Connection", "keep-alive")
		}

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_chat_completion_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		model := c.GetString("model")

		if res.StatusCode == http.StatusOK && !isStreaming {
			dur := time.Now().Sub(start)
			stats.Timing("bricksllm.proxy.get_chat_completion_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			var cost float64 = 0
			chatRes := &goopenai.ChatCompletionResponse{}
			stats.Incr("bricksllm.proxy.get_chat_completion_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_chat_completion_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling openai http chat completion response body", prod, cid, err)
			}

			if err == nil {
				logChatCompletionResponse(log, prod, private, cid, chatRes)
				cost, err = e.EstimateTotalCost(model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost", prod, cid, err)
				}

				// micros := int64(cost * 1000000)
				// err = r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
				// if err != nil {
				// 	stats.Incr("bricksllm.proxy.get_chat_completion_handler.record_key_spend_error", nil, 1)
				// 	logError(log, "error when recording openai spend", prod, cid, err)
				// }
			}

			c.Set("costInUsd", cost)
			c.Set("promptTokenCount", chatRes.Usage.PromptTokens)
			c.Set("completionTokenCount", chatRes.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Now().Sub(start)
			stats.Timing("bricksllm.proxy.get_chat_completion_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_chat_completion_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			logAnthropicErrorResponse(log, bytes, prod, cid)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		// var totalCost float64 = 0
		// var totalTokens int = 0
		content := ""
		defer func() {
			c.Set("content", content)

			// tks, cost, err := e.EstimateChatCompletionStreamCostWithTokenCounts(model, content)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_chat_completion_cost_and_tokens_error", nil, 1)
			// 	logError(log, "error when estimating chat completion stream cost with token counts", prod, cid, err)
			// }

			// estimatedPromptCost := c.GetFloat64("estimatedPromptCostInUsd")
			// totalCost = cost + estimatedPromptCost
			// totalTokens += tks

			// c.Set("costInUsd", totalCost)
			// c.Set("completionTokenCount", totalTokens)
		}()

		stats.Incr("bricksllm.proxy.get_chat_completion_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_chat_completion_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from openai chat completion response", prod, cid, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_chat_completion_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from openai chat completion response", prod, cid, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_chat_completion_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for openai streaming chat completion error response", prod, cid, err)
					return true
				}

				c.SSEvent("", string(bytes))
				return true
			}

			noSpaceLine := bytes.TrimSpace(raw)
			if !bytes.HasPrefix(noSpaceLine, headerData) {
				return true
			}

			noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
			c.SSEvent("", " "+string(noPrefixLine))

			if string(noPrefixLine) == "[DONE]" {
				return false
			}

			chatCompletionStreamResp := &goopenai.ChatCompletionStreamResponse{}
			err = json.Unmarshal(noPrefixLine, chatCompletionStreamResp)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_chat_completion_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling openai chat completion stream response", prod, cid, err)
			}

			if err == nil {
				if len(chatCompletionStreamResp.Choices) > 0 && len(chatCompletionStreamResp.Choices[0].Delta.Content) != 0 {
					content += chatCompletionStreamResp.Choices[0].Delta.Content
				}
			}

			return true
		})

		stats.Timing("bricksllm.proxy.get_chat_completion_handler.streaming_latency", time.Now().Sub(start), nil, 1)
	}
}

func (ps *ProxyServer) Run() {
	go func() {
		ps.log.Info("proxy server listening at 8002")

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
		ps.log.Info("PORT 8002 | POSt   | /api/providers/openai/v1/threads/:thread_id is ready for modifying an openai thread")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id is ready for deleting an openai thread")

		// messages
		ps.log.Info("PORT 8002 | POST   | /api/providers/openai/v1/threads/:thread_id/messages is ready for creating an openai message")
		ps.log.Info("PORT 8002 | GET    | /api/providers/openai/v1/threads/:thread_id/messages/:message_id is ready for retrieving an openai message")
		ps.log.Info("PORT 8002 | POSt   | /api/providers/openai/v1/threads/:thread_id/messages/:message_id is ready for modifying an openai message")
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

		// custom provider
		ps.log.Info("PORT 8002 | POST   | /api/custom/providers/:provider/*wildcard is ready for forwarding requests to custom providers")

		// custom route
		ps.log.Info("PORT 8002 | POST   | /api/routes/*route is ready for forwarding requests to a custom route")

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
