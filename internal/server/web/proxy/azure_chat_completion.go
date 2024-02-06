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

	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

func buildAzureUrl(path, deploymentId, apiVersion, resourceName string) string {
	if path == "/api/providers/azure/openai/deployments/:deployment_id/chat/completions" {
		return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/chat/completions?api-version=%s", resourceName, deploymentId, apiVersion)
	}

	return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/embeddings?api-version=%s", resourceName, deploymentId, apiVersion)
}

func getAzureChatCompletionHandler(r recorder, prod, private bool, psm ProviderSettingsManager, client http.Client, kms keyMemStorage, log *zap.Logger, aoe azureEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(correlationId)

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildAzureUrl(c.FullPath(), c.Param("deployment_id"), c.Query("api-version"), c.GetString("resourceName")), c.Request.Body)
		if err != nil {
			logError(log, "error when creating azure openai http request", prod, cid, err)
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
			stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.http_client_error", nil, 1)
			logError(log, "error when sending chat completion http request to azure openai", prod, cid, err)
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
			dur := time.Now().Sub(start)
			stats.Timing("bricksllm.proxy.get_azure_chat_completion_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading azure openai chat completion response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai response body")
				return
			}

			var cost float64 = 0
			chatRes := &goopenai.ChatCompletionResponse{}
			stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_azure_chat_completion_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling azure openai http chat completion response body", prod, cid, err)
			}

			if err == nil {
				c.Set("model", chatRes.Model)

				logChatCompletionResponse(log, prod, private, cid, chatRes)
				cost, err = aoe.EstimateTotalCost(chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating azure openai cost", prod, cid, err)
				}

				// micros := int64(cost * 1000000)
				// err = r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
				// if err != nil {
				// 	stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.record_key_spend_error", nil, 1)
				// 	logError(log, "error when recording azure openai spend", prod, cid, err)
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
			stats.Timing("bricksllm.proxy.get_azure_chat_completion_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading azyre openai http chat completion response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai response body")
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

		model := ""
		defer func() {
			if len(model) != 0 {
				c.Set("model", model)
			}

			c.Set("content", content)

			// tks, cost, err := aoe.EstimateChatCompletionStreamCostWithTokenCounts(model, content)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_chat_completion_cost_and_tokens_error", nil, 1)
			// 	logError(log, "error when estimating azure openai chat completion stream cost with token counts", prod, cid, err)
			// }

			// estimatedPromptTokenCounts := c.GetInt("promptTokenCount")
			// promptCost, err := aoe.EstimatePromptCost(model, estimatedPromptTokenCounts)
			// if err != nil {
			// 	stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.estimate_chat_completion_cost_and_tokens_error", nil, 1)
			// 	logError(log, "error when estimating azure openai chat completion stream cost with token counts", prod, cid, err)
			// }

			// totalCost = cost + promptCost
			// totalTokens += tks

			// c.Set("costInUsd", totalCost)
			// c.Set("completionTokenCount", totalTokens)
		}()

		stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from azure openai chat completion response", prod, cid, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from azure openai chat completion response", prod, cid, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.json_marshal_error", nil, 1)
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
				stats.Incr("bricksllm.proxy.get_azure_chat_completion_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling azure openai chat completion stream response", prod, cid, err)
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

		stats.Timing("bricksllm.proxy.get_azure_chat_completion_handler.streaming_latency", time.Now().Sub(start), nil, 1)
	}
}
