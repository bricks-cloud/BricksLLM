package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/route"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
)

type routeManager interface {
	GetRouteFromMemDb(path string) *route.Route
}

type cache interface {
	StoreBytes(key string, value []byte, ttl time.Duration) error
	GetBytes(key string) ([]byte, error)
}

func getRouteHandler(prod bool, ca cache, aoe azureEstimator, e estimator, client http.Client, rec recorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		trueStart := time.Now()

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

		raw, exists = c.Get("route_config")
		rc, ok := raw.(*route.Route)
		if !exists || !ok {
			stats.Incr("bricksllm.proxy.get_route_handeler.route_config_not_found", tags, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] route config not found")
			return
		}

		cacheKey := c.GetString("cache_key")
		shouldCache := len(cacheKey) != 0

		if shouldCache {
			bytes, err := ca.GetBytes(cacheKey)
			if err == nil && len(bytes) != 0 {
				stats.Incr("bricksllm.proxy.get_route_handeler.success", nil, 1)
				stats.Timing("bricksllm.proxy.get_route_handeler.success_latency", time.Since(trueStart), nil, 1)

				c.Set("provider", "cached")
				c.Data(http.StatusOK, "application/json", bytes)
				return
			}
		}

		raw, exists = c.Get("settings")
		settings, ok := raw.([]*provider.Setting)
		if !exists || !ok {
			stats.Incr("bricksllm.proxy.get_route_handeler.provider_settings_not_found", tags, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] provider settings not found")
			return
		}

		settingsMap := map[string]*provider.Setting{}
		for _, setting := range settings {
			settingsMap[setting.Id] = setting
		}

		start := time.Now()

		cid := c.GetString(util.STRING_CORRELATION_ID)
		rreq := &route.Request{
			Settings:      settingsMap,
			Key:           kc,
			Client:        client,
			Forwarded:     c.Request,
			CustomId:      c.GetString("customId"),
			Start:         c.GetTime("startTime"),
			UserId:        c.GetString("userId"),
			PolicyId:      c.GetString("policyId"),
			Action:        c.GetString("action"),
			CorrelationId: cid,
		}

		val, exists := c.Get("requestBytes")
		bs, bok := val.([]byte)
		if exists && bok {
			rreq.Request = bs
		}

		runRes, err := rc.RunSteps(rreq, rec, log)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				stats.Incr("bricksllm.proxy.get_route_handeler.timeout", tags, 1)
				logError(log, "running steps time out", prod, err)
				JSON(c, http.StatusRequestTimeout, "[BricksLLM] request timeout")
				return
			}

			stats.Incr("bricksllm.proxy.get_route_handeler.run_steps_error", tags, 1)
			logError(log, "error when running steps", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] cannot run route steps")
			return
		}

		defer runRes.Cancel()

		c.Set("model", runRes.Model)
		c.Set("provider", runRes.Provider)

		res := runRes.Response
		defer res.Body.Close()

		dur := time.Since(start)
		stats.Timing("bricksllm.proxy.get_route_handeler.latency", dur, nil, 1)

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading route response body", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read route response body")
			return
		}

		if res.StatusCode == http.StatusOK {
			stats.Incr("bricksllm.proxy.get_route_handeler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_route_handeler.success_latency", dur, nil, 1)

			if shouldCache && rc.CacheConfig != nil {
				parsed, err := time.ParseDuration(rc.CacheConfig.Ttl)
				if err != nil {
					logError(log, "error when parsing cache config ttl", prod, err)
				}

				if err == nil {
					err := ca.StoreBytes(cacheKey, bytes, parsed)
					if err != nil {
						logError(log, "error when storing cached response", prod, err)
					}
				}

			}

			err := parseResult(c, rc.ShouldRunEmbeddings(), bytes, e, aoe, runRes.Model, runRes.Provider)
			if err != nil {
				logError(log, "error when parsing run steps result", prod, err)
			}
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_azure_embeddings_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_azure_embeddings_handler.error_response", nil, 1)

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

func parseResult(c *gin.Context, runEmbeddings bool, bytes []byte, e estimator, aoe azureEstimator, model, provider string) error {
	base64ChatRes := &EmbeddingResponseBase64{}
	chatRes := &EmbeddingResponse{}

	var cost float64 = 0
	promptTokenCounts := 0
	completionTokenCounts := 0

	defer func() {
		c.Set("provider", provider)
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

		if strings.HasPrefix(strings.ToLower(provider), "azure") {
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

		// micros := int64(cost * 1000000)

		// err := r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
		// if err != nil {
		// 	return err
		// }
	}

	if !runEmbeddings {
		chatRes := &goopenai.ChatCompletionResponse{}
		err := json.Unmarshal(bytes, chatRes)
		if err != nil {
			return err
		}

		promptTokenCounts = chatRes.Usage.PromptTokens
		completionTokenCounts = chatRes.Usage.CompletionTokens

		if strings.HasPrefix(strings.ToLower(provider), "azure") {
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

		// micros := int64(cost * 1000000)
		// err = r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
		// if err != nil {
		// 	return err
		// }
	}

	return nil
}
