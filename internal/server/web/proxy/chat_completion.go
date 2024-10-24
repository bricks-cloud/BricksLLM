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

	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
)

func getChatCompletionHandler(prod, private bool, client http.Client, e estimator) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.GetDuration("requestTimeout"))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai http request")
			return
		}

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		isStreaming := c.GetBool("stream")
		if isStreaming {
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Connection", "keep-alive")
		}

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.http_client_error", nil, 1)

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

		model := c.GetString("model")

		if res.StatusCode == http.StatusOK && !isStreaming {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_chat_completion_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			var cost float64 = 0
			chatRes := &goopenai.ChatCompletionResponse{}
			telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_chat_completion_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling openai http chat completion response body", prod, err)
			}

			if err == nil {
				logChatCompletionResponse(log, prod, private, chatRes)
				cost, err = e.EstimateTotalCost(model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost", prod, err)
				}

				m, exists := c.Get("cost_map")
				if exists {
					converted, ok := m.(*provider.CostMap)
					if ok {
						newCost, err := provider.EstimateTotalCostWithCostMaps(model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens, 1000, converted.PromptCostPerModel, converted.CompletionCostPerModel)
						if err != nil {
							logError(log, "error when estimating openai chat completions total cost with cost maps", prod, err)
							telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_total_cost_with_cost_maps_error", nil, 1)
						}

						if newCost != 0 {
							cost = newCost
						}
					}
				}
			}

			c.Set("costInUsd", cost)
			c.Set("promptTokenCount", chatRes.Usage.PromptTokens)
			c.Set("completionTokenCount", chatRes.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_chat_completion_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.error_response", nil, 1)

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
			return
		}

		buffer := bufio.NewReader(res.Body)
		content := ""
		streamingResponse := [][]byte{}
		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))

		}()

		telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from openai chat completion response", prod, err)

					return false
				}

				telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from openai chat completion response", prod, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for openai streaming chat completion error response", prod, err)
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
				telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling openai chat completion stream response", prod, err)
			}

			if err == nil {
				if len(chatCompletionStreamResp.Choices) > 0 && len(chatCompletionStreamResp.Choices[0].Delta.Content) != 0 {
					content += chatCompletionStreamResp.Choices[0].Delta.Content
				}
			}

			return true
		})

		telemetry.Timing("bricksllm.proxy.get_chat_completion_handler.streaming_latency", time.Since(start), nil, 1)
	}
}
