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

	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func countTokensFromJson(bytes []byte, contentLoc string) (int, error) {
	content := getContentFromJson(bytes, contentLoc)
	return custom.Count(content)
}

func getContentFromJson(bytes []byte, contentLoc string) string {
	result := gjson.Get(string(bytes), contentLoc)
	content := ""

	if len(result.Str) != 0 {
		content += result.Str
	}

	if result.IsArray() {
		for _, val := range result.Array() {
			if len(val.Str) != 0 {
				content += val.Str
			}
		}
	}

	return content
}

type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error *Error `json:"error"`
}

func getCustomProviderHandler(prod, private bool, psm ProviderSettingsManager, cpm CustomProvidersManager, client http.Client, log *zap.Logger, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		tags := []string{
			fmt.Sprintf("path:%s", c.FullPath()),
		}

		stats.Incr("bricksllm.proxy.get_custom_provider_handler.requests", tags, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		raw, exists := c.Get("route_config")
		rc, ok := raw.(*custom.RouteConfig)
		if !exists || !ok {
			stats.Incr("bricksllm.proxy.get_custom_provider_handler.route_config_not_found", tags, 1)
			JSON(c, http.StatusNotFound, "[BricksLLM] requested route config is not found")
			return
		}

		cid := c.GetString(correlationId)
		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logError(log, "error when reading request body", prod, cid, err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, rc.TargetUrl, io.NopCloser(bytes.NewReader(body)))
		if err != nil {
			logError(log, "error when creating custom provider http request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create custom provider http request")
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
			stats.Incr("bricksllm.proxy.get_custom_provider_handler.http_client_error", tags, 1)

			logError(log, "error when sending custom provider request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send custom provider request")
			return
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusOK && !isStreaming {
			dur := time.Now().Sub(start)
			stats.Timing("bricksllm.proxy.get_custom_provider_handler.latency", dur, tags, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading custom provider response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read custom provider response body")
				return
			}

			tks, err := countTokensFromJson(bytes, rc.ResponseCompletionLocation)
			if err != nil {
				logError(log, "error when counting tokens for custom provider completion response", prod, cid, err)
			}

			c.Set("completionTokenCount", tks)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			stats.Timing("bricksllm.proxy.get_custom_provider_handler.error_latency", time.Now().Sub(start), nil, 1)
			stats.Incr("bricksllm.proxy.get_custom_provider_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading custom provider response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read custom provider response body")
				return
			}

			logError(log, "error response from the custom provider", prod, cid, errors.New(string(bytes)))
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		aggregated := ""
		defer func() {
			tks, err := custom.Count(aggregated)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_custom_provider_handler.count_error", nil, 1)
				logError(log, "error when counting tokens for custom provider streaming response", prod, cid, err)
			}

			c.Set("completionTokenCount", tks)
		}()

		stats.Incr("bricksllm.proxy.get_custom_provider_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_custom_provider_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from custom provider response", prod, cid, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_custom_provider_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from custom provider response", prod, cid, err)

				apiErr := &ErrorResponse{
					Error: &Error{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_custom_provider_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for custom provider streaming error response", prod, cid, err)
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

			if string(noPrefixLine) == rc.StreamEndWord {
				return false
			}

			content := getContentFromJson(noPrefixLine, rc.StreamResponseCompletionLocation)
			aggregated += content

			return true
		})

		stats.Timing("bricksllm.proxy.get_custom_provider_handler.streaming_latency", time.Now().Sub(start), nil, 1)
	}
}
