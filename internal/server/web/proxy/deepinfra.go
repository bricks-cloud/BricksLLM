package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

func getDeepinfraCompletionsHandler(prod, private bool, client http.Client, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(logFiledNameCorrelationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepinfra.com/v1/openai/completions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating deepinfra http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create deepinfra http request")
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
			stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to deepinfra", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to deepinfra")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK && !isStreaming {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_deepinfra_completions_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading deepinfra http completions response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read deepinfra response body")
				return
			}

			cr := &goopenai.CompletionResponse{}
			stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_deepinfra_completions_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, cr)
			if err != nil {
				logError(log, "error when unmarshalling deepinfra http chat completion response body", prod, err)
			}

			if err == nil {
				logVllmCompletionResponse(log, cr, prod, private, cid)
			}

			c.Set("promptTokenCount", cr.Usage.PromptTokens)
			c.Set("completionTokenCount", cr.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_deepinfra_completions_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading deepinfra http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read deepinfra response body")
				return
			}

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling deepinfra chat completion error response body", prod, err)
			}

			logOpenAiError(log, prod, cid, errorRes)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		content := ""
		streamingResponse := [][]byte{}
		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from deepinfra completions response", prod, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from deepinfra completions response", prod, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for deepinfra streaming chat completion error response", prod, err)
					return false
				}

				c.SSEvent("", string(bytes))
				c.SSEvent("", " [DONE]")
				return false
			}

			streamingResponse = append(streamingResponse, raw)

			noSpaceLine := bytes.TrimSpace(raw)
			if !bytes.HasPrefix(noSpaceLine, headerData) {
				return true
			}

			noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
			c.SSEvent("", " "+string(noPrefixLine))

			if string(noPrefixLine) == "[DONE]" {
				return false
			}

			completionsStreamResp := &goopenai.CompletionResponse{}
			err = json.Unmarshal(noPrefixLine, completionsStreamResp)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_deepinfra_completions_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling deepinfra completions stream response", prod, err)
			}

			if err == nil {
				if len(completionsStreamResp.Choices) > 0 && len(completionsStreamResp.Choices[0].Text) != 0 {
					content += completionsStreamResp.Choices[0].Text
				}
			}

			return true
		})

		stats.Timing("bricksllm.proxy.get_deepinfra_completions_handler.streaming_latency", time.Since(start), nil, 1)

	}
}

func getDeepinfraChatCompletionsHandler(prod, private bool, client http.Client, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(logFiledNameCorrelationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepinfra.com/v1/openai/chat/completions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating deepinfra chat completions http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create deepinfra http request")
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
			stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to deepinfra", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to deepinfra")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		if res.StatusCode == http.StatusOK && !isStreaming {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_deepinfra_chat_completions_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading vllm chat completions response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read vllm response body")
				return
			}

			chatRes := &goopenai.ChatCompletionResponse{}
			stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_deepinfra_chat_completions_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling deepinfra chat completions response body", prod, err)
			}

			if err == nil {
				logChatCompletionResponse(log, prod, private, cid, chatRes)
			}

			c.Set("promptTokenCount", chatRes.Usage.PromptTokens)
			c.Set("completionTokenCount", chatRes.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_deepinfra_chat_completions_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading deepinfra chat completions response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read deepinfras response body")
				return
			}

			logAnthropicErrorResponse(log, bytes, prod, cid)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		content := ""
		streamingResponse := [][]byte{}
		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from deepinfra chat completions response", prod, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from deepinfra chat completions response", prod, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for streaming deepinfra chat completions error response", prod, err)
					return false
				}

				c.SSEvent("", string(bytes))
				c.SSEvent("", " [DONE]")
				return false
			}

			streamingResponse = append(streamingResponse, raw)

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
				stats.Incr("bricksllm.proxy.get_deepinfra_chat_completions_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling deepinfra chat completions stream response", prod, err)
			}

			if err == nil {
				if len(chatCompletionStreamResp.Choices) > 0 && len(chatCompletionStreamResp.Choices[0].Delta.Content) != 0 {
					content += chatCompletionStreamResp.Choices[0].Delta.Content
				}
			}

			return true
		})

		stats.Timing("bricksllm.proxy.get_deepinfra_chat_completions_handler.streaming_latency", time.Since(start), nil, 1)
	}
}

func getDeepinfraEmbeddingsHandler(prod, private bool, client http.Client, log *zap.Logger, e deepinfraEstimator, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.proxy.get_deepinfra_embeddings_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(logFiledNameCorrelationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepinfra.com/v1/openai/embeddings", c.Request.Body)
		if err != nil {
			logError(log, "error when creating deepinfra embeddings http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create deepinfra http request")
			return
		}

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.proxy.get_deepinfra_embeddings_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to deepinfra", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to deepinfra")
			return
		}

		defer res.Body.Close()

		dur := time.Since(start)
		stats.Timing("bricksllm.proxy.get_deepinfra_embeddings_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading deepinfra embedding response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read deepinfra openai embedding response body")
			return
		}

		var cost float64 = 0
		chatRes := &EmbeddingResponse{}
		promptTokenCounts := 0

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_deepinfra_embeddings_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_deepinfra_embeddings_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling deepinfra openai embedding response body", prod, err)
			}

			model := c.GetString("model")

			totalTokens := 0
			if err == nil {
				logEmbeddingResponse(log, prod, private, cid, chatRes)
				totalTokens = chatRes.Usage.TotalTokens
				promptTokenCounts = chatRes.Usage.PromptTokens

				cost, err = e.EstimateEmbeddingsInputCost(model, totalTokens)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_deepinfra_embeddings_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating azure openai cost for embedding", prod, err)
				}
			}
		}

		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", promptTokenCounts)

		if res.StatusCode != http.StatusOK {
			stats.Incr("bricksllm.proxy.get_deepinfra_embeddings_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling deepinfra openai embedding error response body", prod, err)
			}

			logOpenAiError(log, prod, cid, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}
