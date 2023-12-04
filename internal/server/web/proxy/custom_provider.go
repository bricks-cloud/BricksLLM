package proxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func countTokensFromJson(bytes []byte, contentLoc string) (int, error) {
	totalTokens := 0
	result := gjson.Get(string(bytes), contentLoc)
	contents := []string{}

	if len(result.Str) != 0 {
		contents = append(contents, result.Str)
	}

	if result.IsArray() {
		for _, val := range result.Array() {
			if len(val.Str) != 0 {
				contents = append(contents, val.Str)
			}
		}
	}

	for _, content := range contents {
		tks, err := custom.Count(content)
		if err != nil {
			return 0, nil
		}

		totalTokens += tks
	}

	if totalTokens == 0 {
		return 0, fmt.Errorf("content location %s does not have any value", contentLoc)
	}

	return totalTokens, nil
}

func getCustomProviderHandler(prod, private bool, psm ProviderSettingsManager, cpm CustomProvidersManager, client http.Client, log *zap.Logger, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		tags := []string{
			fmt.Sprintf("path:%s", c.FullPath()),
		}

		stats.Incr("bricksllm.web.get_custom_provider_handler.requests", tags, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		raw, exists := c.Get("key")
		kc, ok := raw.(*key.ResponseKey)
		if !exists || !ok {
			stats.Incr("bricksllm.web.get_custom_provider_handler.api_key_not_registered", tags, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
			return
		}

		raw, exists = c.Get("provider")
		cp, ok := raw.(*custom.Provider)
		if !exists || !ok {
			stats.Incr("bricksllm.web.get_custom_provider_handler.provider_not_found", tags, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] requested custom provider is not found")
			return
		}

		raw, exists = c.Get("route_config")
		rc, ok := raw.(*custom.RouteConfig)
		if !exists || !ok {
			stats.Incr("bricksllm.web.get_custom_provider_handler.route_config_not_found", tags, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] requested route config is not found")
			return
		}

		cid := c.GetString(correlationId)
		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, rc.TargetUrl, c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		req.Header.Set("Content-Type", "application/json")

		err = setAuthenticationHeader(psm, req, kc.SettingId, cp.AuthenticationParam)
		if err != nil {
			stats.Incr("bricksllm.web.get_pass_through_handler.set_authentication_header_error", tags, 1)
			logError(log, "error when setting http request authentication header", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] error when setting authentication header")
			return
		}

		isStreaming := c.GetBool("streaming")
		if isStreaming {
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Connection", "keep-alive")
		}

		start := time.Now()
		res, err := client.Do(req)
		if err != nil {
			stats.Incr("bricksllm.web.get_custom_provider_handler.http_client_error", tags, 1)

			logError(log, "error when sending custom provider request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send custom provider request")
			return
		}
		defer res.Body.Close()

		if !isStreaming {
			dur := time.Now().Sub(start)
			stats.Timing("bricksllm.web.get_custom_provider_handler.latency", dur, tags, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading custom provider response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read custom provider response body")
				return
			}

			if res.StatusCode == http.StatusOK {
				tks, err := countTokensFromJson(bytes, rc.ResponseCompletionLocation)
				if err != nil {
					logError(log, "error when counting tokens for custom provider completion response", prod, cid, err)
				}

				c.Set("completionTokenCount", tks)
			}

			if res.StatusCode != http.StatusOK {
				stats.Timing("bricksllm.web.get_custom_provider_handler.error_latency", dur, nil, 1)
				stats.Incr("bricksllm.web.get_custom_provider_handler.error_response", nil, 1)

				logError(log, "error response from the custom provider", prod, cid, errors.New(string(bytes)))
			}

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		c.Stream(func(w io.Writer) bool {
			var totalTokens int = 0

			for {
				raw, err := buffer.ReadBytes('\n')
				if err != nil {
					logError(log, "error when unmarshalling openai http error response body", prod, cid, err)
					c.SSEvent("", fmt.Sprintf(`{"error": "%v"}`, err))

					if err == io.EOF {
						return false
					}
					continue
				}

				noSpaceLine := bytes.TrimSpace(raw)
				if bytes.HasPrefix(noSpaceLine, errorPrefix) {
					noErrorPreixLine := bytes.TrimPrefix(noSpaceLine, errorPrefix)
					c.SSEvent("", fmt.Sprintf(`{"error": "%s"}`, noErrorPreixLine))
					continue
				}

				if !bytes.HasPrefix(noSpaceLine, headerData) {
					continue
				}

				noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
				c.SSEvent("", " "+string(noPrefixLine))

				if string(noPrefixLine) == rc.StreamEndWord {
					break
				}

				tks, err := countTokensFromJson(noPrefixLine, rc.StreamResponseCompletionLocation)
				if err != nil {
					logError(log, "error when counting tokens for custom provider streaming completion responses", prod, cid, err)
				}

				totalTokens += tks
			}

			c.Set("completionTokenCount", totalTokens)

			return false
		})
	}
}
