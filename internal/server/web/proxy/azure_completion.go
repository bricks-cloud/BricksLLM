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
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logAzureCompletionsRequest(log *zap.Logger, prod, private bool, cr *goopenai.CompletionRequest) {
	if prod {
		fields := []zapcore.Field{
			zap.String("model", cr.Model),
			zap.Any("prompt", cr.Prompt),
			zap.Any("suffix", cr.Suffix),
			zap.Int("max_tokens", cr.MaxTokens),
			zap.Float32("temperature", cr.Temperature),
			zap.Float32("top_p", cr.TopP),
			zap.Int("n", cr.N),
			zap.Bool("stream", cr.Stream),
			zap.Int("logprobs", cr.LogProbs),
			zap.Bool("echo", cr.Echo),
			zap.Any("stop", cr.Stop),
			zap.Float32("presence_penalty", cr.PresencePenalty),
			zap.Float32("frequency_penalty", cr.FrequencyPenalty),
			zap.Int("best_of", cr.BestOf),
			zap.Any("logit_bias", cr.LogitBias),
			zap.String("user", cr.User),
		}

		if !private && cr.Prompt != nil {
			fields = append(fields, zap.Any("prompt", cr.Prompt))
		}

		log.Info("azure openai create completions request", fields...)
	}
}

func logAzureCompletionsResponse(log *zap.Logger, prod, private bool, cr *goopenai.CompletionResponse) {
	if prod {
		fields := []zapcore.Field{
			zap.String("id", cr.ID),
			zap.String("object", cr.Object),
			zap.Int64("created", cr.Created),
			zap.String("model", cr.Model),
			zap.Any("usage", cr.Usage),
		}

		if !private && len(cr.Choices) != 0 {
			fields = append(fields, zap.Any("chocies", cr.Choices))
		}

		log.Info("azure openai create completions response", fields...)
	}
}

func getAzureCompletionsHandler(prod, private bool, client http.Client, aoe azureEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		stats.Incr("bricksllm.proxy.get_azure_completions_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildAzureUrl(c.FullPath(), c.Param("deployment_id"), c.Query("api-version"), c.GetString("resourceName")), c.Request.Body)
		if err != nil {
			logError(log, "error when creating azure openai completions http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create azure openai completions http request")
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
			stats.Incr("bricksllm.proxy.get_azure_completions_handler.http_client_error", nil, 1)
			logError(log, "error when sending completions http request to azure openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send completions request to azure openai")
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
			stats.Timing("bricksllm.proxy.get_azure_completions_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading azure openai completions response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai completions response body")
				return
			}

			var cost float64 = 0
			cr := &goopenai.CompletionResponse{}
			stats.Incr("bricksllm.proxy.get_azure_completions_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_azure_completions_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, cr)
			if err != nil {
				logError(log, "error when unmarshalling azure openai http completions response body", prod, err)
			}

			if err == nil {
				c.Set("model", cr.Model)

				logAzureCompletionsResponse(log, prod, private, cr)
				cost, err = aoe.EstimateTotalCost(cr.Model, cr.Usage.PromptTokens, cr.Usage.CompletionTokens)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_azure_completions_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating azure openai cost", prod, err)
				}

				m, exists := c.Get("cost_map")
				if exists {
					converted, ok := m.(*provider.CostMap)
					if ok {
						newCost, err := provider.EstimateTotalCostWithCostMaps(cr.Model, cr.Usage.PromptTokens, cr.Usage.CompletionTokens, 1000, converted.PromptCostPerModel, converted.CompletionCostPerModel)
						if err != nil {
							logError(log, "error when estimating azure completions total cost with cost maps", prod, err)
							stats.Incr("bricksllm.proxy.get_azure_completions_handler.estimate_total_cost_with_cost_maps_error", nil, 1)
						}

						if newCost != 0 {
							cost = newCost
						}
					}
				}
			}

			c.Set("costInUsd", cost)
			c.Set("promptTokenCount", cr.Usage.PromptTokens)
			c.Set("completionTokenCount", cr.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_azure_completions_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_azure_completions_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading azyre openai http completions response body", prod, err)
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
		}()

		stats.Incr("bricksllm.proxy.get_azure_completions_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_azure_completions_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from azure openai completions response", prod, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_azure_completions_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from azure openai completions response", prod, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_azure_completions_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for openai streaming completions error response", prod, err)
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

			completionsStreamResp := &goopenai.CompletionResponse{}
			err = json.Unmarshal(noPrefixLine, completionsStreamResp)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_azure_completions_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling azure openai completions stream response", prod, err)
			}

			if len(model) == 0 && len(completionsStreamResp.Model) != 0 {
				model = completionsStreamResp.Model
			}

			if err == nil {
				if len(completionsStreamResp.Choices) > 0 && len(completionsStreamResp.Choices[0].Text) != 0 {
					content += completionsStreamResp.Choices[0].Text
				}
			}

			return true
		})

		stats.Timing("bricksllm.proxy.get_azure_completions_handler.streaming_latency", time.Since(start), nil, 1)
	}
}
