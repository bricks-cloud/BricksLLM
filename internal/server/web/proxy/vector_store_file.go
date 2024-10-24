package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
)

func getCreateVectorStoreFileHandler(prod bool, client http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_create_vector_store_file_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.GetDuration("requestTimeout"))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/vector_stores/"+c.Param("vector_store_id")+"/files", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai http request")
			return
		}

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_create_vector_store_file_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_create_vector_store_file_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			telemetry.Incr("bricksllm.proxy.get_create_vector_store_file_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_create_vector_store_file_handler.success_latency", dur, nil, 1)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_create_vector_store_file_handler.error_latency", dur, nil, 1)
		telemetry.Incr("bricksllm.proxy.get_create_vector_store_file_handler.error_response", nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai http chat completion response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		errorRes := &goopenai.ErrorResponse{}
		err = json.Unmarshal(bytes, errorRes)
		if err != nil {
			logError(log, "error when unmarshalling openai chat completion error response body", prod, err)
		}

		logOpenAiError(log, prod, errorRes)

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func getListVectorStoreFilesHandler(prod bool, client http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_list_vector_store_files_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.GetDuration("requestTimeout"))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/vector_stores/"+c.Param("vector_store_id")+"/files", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai http request")
			return
		}

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_list_vector_store_files_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_list_vector_store_files_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			telemetry.Incr("bricksllm.proxy.get_list_vector_store_files_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_list_vector_store_files_handler.success_latency", dur, nil, 1)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_list_vector_store_files_handler.error_latency", dur, nil, 1)
		telemetry.Incr("bricksllm.proxy.get_list_vector_store_files_handler.error_response", nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai http chat completion response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		errorRes := &goopenai.ErrorResponse{}
		err = json.Unmarshal(bytes, errorRes)
		if err != nil {
			logError(log, "error when unmarshalling openai chat completion error response body", prod, err)
		}

		logOpenAiError(log, prod, errorRes)

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func getGetVectorStoreFileHandler(prod bool, client http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_get_vector_store_file_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.GetDuration("requestTimeout"))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/vector_stores/"+c.Param("vector_store_id")+"/files/"+c.Param("file_id"), c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai http request")
			return
		}

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_get_vector_store_file_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_get_vector_store_file_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			telemetry.Incr("bricksllm.proxy.get_get_vector_store_file_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_get_vector_store_file_handler.success_latency", dur, nil, 1)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_get_vector_store_file_handler.error_latency", dur, nil, 1)
		telemetry.Incr("bricksllm.proxy.get_get_vector_store_file_handler.error_response", nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai http chat completion response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		errorRes := &goopenai.ErrorResponse{}
		err = json.Unmarshal(bytes, errorRes)
		if err != nil {
			logError(log, "error when unmarshalling openai chat completion error response body", prod, err)
		}

		logOpenAiError(log, prod, errorRes)

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func getDeleteVectorStoreFileHandler(prod bool, client http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_delete_vector_store_file_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.GetDuration("requestTimeout"))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/vector_stores/"+c.Param("vector_store_id")+"/files/"+c.Param("file_id"), c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai http request")
			return
		}

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_delete_vector_store_file_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_delete_vector_store_file_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			telemetry.Incr("bricksllm.proxy.get_delete_vector_store_file_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_delete_vector_store_file_handler.success_latency", dur, nil, 1)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_delete_vector_store_file_handler.error_latency", dur, nil, 1)
		telemetry.Incr("bricksllm.proxy.get_delete_vector_store_file_handler.error_response", nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai http chat completion response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		errorRes := &goopenai.ErrorResponse{}
		err = json.Unmarshal(bytes, errorRes)
		if err != nil {
			logError(log, "error when unmarshalling openai chat completion error response body", prod, err)
		}

		logOpenAiError(log, prod, errorRes)

		c.Data(res.StatusCode, "application/json", bytes)
	}
}
