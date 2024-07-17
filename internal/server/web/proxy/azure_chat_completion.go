package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
)

func buildAzureUrl(path, deploymentId, apiVersion, resourceName string) string {
	if path == "/api/providers/azure/openai/deployments/:deployment_id/chat/completions" {
		return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/chat/completions?api-version=%s", resourceName, deploymentId, apiVersion)
	}

	if path == "/api/providers/azure/openai/deployments/:deployment_id/completions" {
		return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/completions?api-version=%s", resourceName, deploymentId, apiVersion)
	}

	return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/embeddings?api-version=%s", resourceName, deploymentId, apiVersion)
}

func getAzureChatCompletionHandler(prod, private bool, client http.Client, aoe azureEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildAzureUrl(c.FullPath(), c.Param("deployment_id"), c.Query("api-version"), c.GetString("resourceName")), c.Request.Body)
		if err != nil {
			logError(log, "error when creating azure openai http request", prod, err)
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
			telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.http_client_error", nil, 1)
			logError(log, "error when sending chat completion http request to azure openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send chat completion request to azure openai")
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
			telemetry.Timing("bricksllm.proxy.get_azure_chat_completion_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading azure openai chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai response body")
				return
			}

			var cost float64 = 0
			chatRes := &goopenai.ChatCompletionResponse{}
			telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_azure_chat_completion_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling azure openai http chat completion response body", prod, err)
			}

			if err == nil {
				c.Set("model", chatRes.Model)

				logChatCompletionResponse(log, prod, private, chatRes)
				cost, err = aoe.EstimateTotalCost(chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating azure openai cost", prod, err)
				}

				m, exists := c.Get("cost_map")
				if exists {
					converted, ok := m.(*provider.CostMap)
					if ok {
						newCost, err := provider.EstimateTotalCostWithCostMaps(chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens, 1000, converted.PromptCostPerModel, converted.CompletionCostPerModel)
						if err != nil {
							logError(log, "error when estimating azure chat completions total cost with cost maps", prod, err)
							telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_total_cost_with_cost_maps_error", nil, 1)
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
			telemetry.Timing("bricksllm.proxy.get_azure_chat_completion_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading azyre openai http chat completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai response body")
				return
			}

			logAnthropicErrorResponse(log, bytes, prod)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		// var totalCost float64 = 0
		// var totalTokens int = 0
		content := ""

		model := ""
		defer func() {
			if len(model) != 0 {
				c.Set("model", model)
			}

			c.Set("content", content)

			// tks, cost, err := aoe.EstimateChatCompletionStreamCostWithTokenCounts(model, content)
			// if err != nil {
			// 	telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_chat_completion_cost_and_tokens_error", nil, 1)
			// 	logError(log, "error when estimating azure openai chat completion stream cost with token counts", prod, err)
			// }

			// estimatedPromptTokenCounts := c.GetInt("promptTokenCount")
			// promptCost, err := aoe.EstimatePromptCost(model, estimatedPromptTokenCounts)
			// if err != nil {
			// 	telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_chat_completion_cost_and_tokens_error", nil, 1)
			// 	logError(log, "error when estimating azure openai chat completion stream cost with token counts", prod, err)
			// }

			// totalCost = cost + promptCost
			// totalTokens += tks

			// c.Set("costInUsd", totalCost)
			// c.Set("completionTokenCount", totalTokens)
		}()

		telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from azure openai chat completion response", prod, err)

					return false
				}

				telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from azure openai chat completion response", prod, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for openai streaming chat completion error response", prod, err)
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
				telemetry.Incr("bricksllm.proxy.get_azure_chat_completion_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling azure openai chat completion stream response", prod, err)
			}

			if len(model) == 0 && len(chatCompletionStreamResp.Model) != 0 {
				model = chatCompletionStreamResp.Model
			}

			if err == nil {
				if len(chatCompletionStreamResp.Choices) > 0 && len(chatCompletionStreamResp.Choices[0].Delta.Content) != 0 {
					content += chatCompletionStreamResp.Choices[0].Delta.Content
				}
			}

			return true
		})

		telemetry.Timing("bricksllm.proxy.get_azure_chat_completion_handler.streaming_latency", time.Since(start), nil, 1)
	}
}
