package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
)

func getAzureEmbeddingsHandler(prod, private bool, client http.Client, aoe azureEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		// raw, exists := c.Get("key")
		// kc, ok := raw.(*key.ResponseKey)
		// if !exists || !ok {
		// 	telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.api_key_not_registered", nil, 1)
		// 	JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
		// 	return
		// }

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, buildAzureUrl(c.FullPath(), c.Param("deployment_id"), c.Query("api-version"), c.GetString("resourceName")), c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		copyHttpHeaders(c.Request, req)

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.http_client_error", nil, 1)

			logError(log, "error when sending embedding request to azure openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send embedding request to azure openai")
			return
		}
		defer res.Body.Close()

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_azure_embeddings_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai embedding response body")
			return
		}

		var cost float64 = 0
		chatRes := &EmbeddingResponse{}
		promptTokenCounts := 0
		base64ChatRes := &EmbeddingResponseBase64{}
		if res.StatusCode == http.StatusOK {
			telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_azure_embeddings_handler.success_latency", dur, nil, 1)

			format := c.GetString("encoding_format")

			if format == "base64" {
				err = json.Unmarshal(bytes, base64ChatRes)
				if err != nil {
					logError(log, "error when unmarshalling azure openai base64 embedding response body", prod, err)
				}
			}

			if format != "base64" {
				err = json.Unmarshal(bytes, chatRes)
				if err != nil {
					logError(log, "error when unmarshalling azure openai embedding response body", prod, err)
				}
			}

			model := c.GetString("model")

			totalTokens := 0
			if err == nil {
				if format == "base64" {
					logBase64EmbeddingResponse(log, prod, private, base64ChatRes)
					totalTokens = base64ChatRes.Usage.TotalTokens
					promptTokenCounts = base64ChatRes.Usage.PromptTokens
				}

				if format != "base64" {
					logEmbeddingResponse(log, prod, private, chatRes)
					totalTokens = chatRes.Usage.TotalTokens
					promptTokenCounts = chatRes.Usage.PromptTokens
				}

				cost, err = aoe.EstimateEmbeddingsInputCost(model, totalTokens)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating azure openai cost for embedding", prod, err)
				}

				m, exists := c.Get("cost_map")
				if exists {
					converted, ok := m.(*provider.CostMap)
					if ok {
						newCost, err := provider.EstimateCostWithCostMap(model, totalTokens, 1000, converted.EmbeddingsCostPerModel)
						if err != nil {
							logError(log, "error when estimating azure embeddings total cost with cost maps", prod, err)
							telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.estimate_cost_with_cost_map_error", nil, 1)
						}

						if newCost != 0 {
							cost = newCost
						}
					}
				}
			}
		}

		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", promptTokenCounts)

		if res.StatusCode != http.StatusOK {
			telemetry.Timing("bricksllm.proxy.get_azure_embeddings_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_azure_embeddings_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling azure openai embedding error response body", prod, err)
			}

			logOpenAiError(log, prod, errorRes)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}
