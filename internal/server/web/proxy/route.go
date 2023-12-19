package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

func getRouteHandler(prod, private bool, psm ProviderSettingsManager, aoe azureEstimator, e estimator, r recorder, cpm CustomProvidersManager, client http.Client, log *zap.Logger, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		tags := []string{
			fmt.Sprintf("path:%s", c.FullPath()),
		}

		stats.Incr("bricksllm.proxy.get_route_handeler.requests", tags, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		raw, exists := c.Get("key")
		kc, ok := raw.(*key.ResponseKey)
		if !exists || !ok {
			stats.Incr("bricksllm.proxy.get_route_handeler.api_key_not_registered", tags, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
			return
		}

		settings := map[string]*provider.Setting{}
		raw, exists = c.Get("route_config")
		rc, ok := raw.(*route.Route)
		if !exists || !ok {
			stats.Incr("bricksllm.proxy.get_route_handeler.route_config_not_found", tags, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] requested route config is not found")
			return
		}

		cid := c.GetString(correlationId)
		start := time.Now()
		runRes, err := rc.RunSteps(&route.Request{
			Settings:  settings,
			Key:       kc,
			Client:    client,
			Forwarded: c.Request,
		})

		if err != nil {
			stats.Incr("bricksllm.proxy.get_route_handeler.run_steps_error", tags, 1)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot run route steps")
			return
		}

		res := runRes.Response
		defer res.Body.Close()

		dur := time.Now().Sub(start)
		stats.Timing("bricksllm.proxy.get_route_handeler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai embedding response body", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read azure openai embedding response body")
			return
		}

		var cost float64 = 0
		promptTokenCounts := 0
		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_route_handeler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_route_handeler.success_latency", dur, nil, 1)

			err := parseResult(c, kc, rc.ShouldRunEmbeddings(), bytes, e, aoe, r, runRes.Model, runRes.Provider)
			if err != nil {
				logError(log, "error when parsing run steps result", prod, cid, err)
			}
		}

		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", promptTokenCounts)

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_azure_embeddings_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_azure_embeddings_handler.error_response", nil, 1)

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling azure openai embedding error response body", prod, cid, err)
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

func parseResult(c *gin.Context, kc *key.ResponseKey, runEmbeddings bool, bytes []byte, e estimator, aoe azureEstimator, r recorder, model, provider string) error {
	base64ChatRes := &EmbeddingResponseBase64{}
	chatRes := &EmbeddingResponse{}

	var cost float64 = 0
	promptTokenCounts := 0
	completionTokenCounts := 0

	defer func() {
		c.Set("costInUsd", cost)
		c.Set("promptTokenCount", promptTokenCounts)
		c.Set("completionTokenCount", completionTokenCounts)
	}()

	if runEmbeddings {
		format := c.GetString("encoding_format")
		if format == "base64" {
			err := json.Unmarshal(bytes, base64ChatRes)
			if err != nil {
				return err
			}
		}

		if format != "base64" {
			err := json.Unmarshal(bytes, chatRes)
			if err != nil {
				return err
			}
		}

		totalTokens := 0
		if format == "base64" {
			totalTokens = base64ChatRes.Usage.TotalTokens
			promptTokenCounts = base64ChatRes.Usage.PromptTokens
		}

		if format != "base64" {
			totalTokens = chatRes.Usage.TotalTokens
			promptTokenCounts = chatRes.Usage.PromptTokens
		}

		if provider == "azure" {
			ecost, err := aoe.EstimateEmbeddingsInputCost(model, totalTokens)
			if err != nil {
				return err
			}

			cost = ecost
		} else if provider == "openai" {
			ecost, err := e.EstimateEmbeddingsInputCost(model, totalTokens)
			if err != nil {
				return err
			}

			cost = ecost
		}

		micros := int64(cost * 1000000)

		err := r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
		if err != nil {
			return err
		}
	}

	if !runEmbeddings {
		chatRes := &goopenai.ChatCompletionResponse{}
		err := json.Unmarshal(bytes, chatRes)
		if err != nil {
			return err
		}

		c.Set("model", chatRes.Model)

		if provider == "azure" {
			cost, err = aoe.EstimateTotalCost(chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
			if err != nil {
				return err
			}

		} else if provider == "openai" {
			cost, err = e.EstimateTotalCost(chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
			if err != nil {
				return err
			}
		}

		micros := int64(cost * 1000000)
		err = r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
		if err != nil {
			return err
		}

		completionTokenCounts = chatRes.Usage.CompletionTokens
	}

	return nil
}
