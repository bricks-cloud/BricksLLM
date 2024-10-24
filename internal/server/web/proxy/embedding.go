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

func getEmbeddingHandler(prod, private bool, client http.Client, e estimator) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_embedding_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		// raw, exists := c.Get("key")
		// kc, ok := raw.(*key.ResponseKey)
		// if !exists || !ok {
		// 	telemetry.Incr("bricksllm.proxy.get_embedding_handler.api_key_not_registered", nil, 1)
		// 	JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
		// 	return
		// }

		ctx, cancel := context.WithTimeout(context.Background(), c.GetDuration("requestTimeout"))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, c.Request.Method, "https://api.openai.com/v1/embeddings", c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		copyHttpHeaders(c.Request, req, c.GetBool("removeUserAgent"))

		start := time.Now()

		res, err := client.Do(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_embedding_handler.http_client_error", nil, 1)

			logError(log, "error when sending embedding request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send embedding request to openai")
			return
		}
		defer res.Body.Close()

		dur := time.Since(start)
		telemetry.Timing("bricksllm.proxy.get_embedding_handler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai embedding response body")
			return
		}

		var cost float64 = 0
		chatRes := &EmbeddingResponse{}
		promptTokenCounts := 0
		base64ChatRes := &EmbeddingResponseBase64{}
		if res.StatusCode == http.StatusOK {
			telemetry.Incr("bricksllm.proxy.get_embedding_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_embedding_handler.success_latency", dur, nil, 1)

			format := c.GetString("encoding_format")

			if format == "base64" {
				err = json.Unmarshal(bytes, base64ChatRes)
				if err != nil {
					logError(log, "error when unmarshalling openai base64 embedding response body", prod, err)
				}
			}

			if format != "base64" {
				err = json.Unmarshal(bytes, chatRes)
				if err != nil {
					logError(log, "error when unmarshalling openai embedding response body", prod, err)
				}
			}

			model := c.GetString("model")

			totalTokens := 0
			if err == nil {
				if format == "base64" {
					logBase64EmbeddingResponse(log, prod, private, base64ChatRes)
					promptTokenCounts = base64ChatRes.Usage.PromptTokens
					totalTokens = base64ChatRes.Usage.TotalTokens
				}

				if format != "base64" {
					logEmbeddingResponse(log, prod, private, chatRes)
					promptTokenCounts = chatRes.Usage.PromptTokens
					totalTokens = chatRes.Usage.TotalTokens
				}

				cost, err = e.EstimateEmbeddingsInputCost(model, totalTokens)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_embedding_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost for embedding", prod, err)
				}

				m, exists := c.Get("cost_map")
				if exists {
					converted, ok := m.(*provider.CostMap)
					if ok {
						newCost, err := provider.EstimateCostWithCostMap(model, totalTokens, 1000, converted.EmbeddingsCostPerModel)
						if err != nil {
							logError(log, "error when estimating openai embeddings total cost with cost maps", prod, err)
							telemetry.Incr("bricksllm.proxy.get_embedding_handler.estimate_cost_with_cost_map_error", nil, 1)
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
			telemetry.Timing("bricksllm.proxy.get_embedding_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_embedding_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai embedding error response body", prod, err)
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
