package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type anthropicEstimator interface {
	EstimateTotalCost(model string, promptTks, completionTks int) (float64, error)
	EstimateCompletionCost(model string, tks int) (float64, error)
	EstimatePromptCost(model string, tks int) (float64, error)
	Count(input string) int
	CountMessagesTokens(messages []anthropic.Message) int
}

func copyHttpHeaders(source *http.Request, dest *http.Request, removeUseAgent bool) {
	for k := range source.Header {
		if strings.ToLower(k) != "X-CUSTOM-EVENT-ID" {
			dest.Header.Set(k, source.Header.Get(k))
		}
	}

	if removeUseAgent {
		dest.Header.Del("User-Agent")
	}

	dest.Header.Set("Accept-Encoding", "*")
}

func getCompletionHandler(prod, private bool, client http.Client, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/complete", c.Request.Body)
		if err != nil {
			logError(log, "error when creating anthropic http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create anthropic http request")
			return
		}

		// raw, exists := c.Get("key")
		// kc, ok := raw.(*key.ResponseKey)
		// if !exists || !ok {
		// 	telemetry.Incr("bricksllm.proxy.get_completion_handler.api_key_not_registered", nil, 1)
		// 	JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
		// 	return
		// }

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
			telemetry.Incr("bricksllm.proxy.get_completion_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to anthropic", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to anthropic")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		// model := c.GetString("model")

		if !isStreaming && res.StatusCode == http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_completion_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading anthropic http completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read anthropic response body")
				return
			}

			// var cost float64 = 0
			// completionTokens := 0
			completionRes := &anthropic.CompletionResponse{}
			telemetry.Incr("bricksllm.proxy.get_completion_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_completion_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, completionRes)
			if err != nil {
				logError(log, "error when unmarshalling anthropic http completion response body", prod, err)
			}

			logCompletionResponse(log, bytes, prod, private)

			c.Set("content", completionRes.Completion)

			// if err == nil {
			// 	logCompletionResponse(log, bytes, prod, private)
			// 	completionTokens = e.Count(completionRes.Completion)
			// 	completionTokens += anthropicCompletionMagicNum
			// 	promptTokens := c.GetInt("promptTokenCount")
			// 	cost, err = e.EstimateTotalCost(model, promptTokens, completionTokens)
			// 	if err != nil {
			// 		telemetry.Incr("bricksllm.proxy.get_completion_handler.estimate_total_cost_error", nil, 1)
			// 		logError(log, "error when estimating anthropic cost", prod, err)
			// 	}

			// 	micros := int64(cost * 1000000)
			// 	err = r.RecordKeySpend(kc.KeyId, micros, kc.CostLimitInUsdUnit)
			// 	if err != nil {
			// 		telemetry.Incr("bricksllm.proxy.get_completion_handler.record_key_spend_error", nil, 1)
			// 		logError(log, "error when recording anthropic spend", prod, err)
			// 	}
			// }

			// c.Set("costInUsd", cost)
			// c.Set("completionTokenCount", completionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_completion_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_completion_handler.error_response", nil, 1)
			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading anthropic http completion response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read anthropic response body")
				return
			}

			logAnthropicErrorResponse(log, bytes, prod)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		// var totalCost float64 = 0

		content := ""
		streamingResponse := [][]byte{}
		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		// defer func() {
		// 	tks := e.Count(content)
		// 	model := c.GetString("model")
		// 	cost, err := e.EstimateCompletionCost(model, tks)
		// 	if err != nil {
		// 		telemetry.Incr("bricksllm.proxy.get_completion_handler.estimate_completion_cost_error", nil, 1)
		// 		logError(log, "error when estimating anthropic completion stream cost", prod, err)
		// 	}

		// 	estimatedPromptCost := c.GetFloat64("estimatedPromptCostInUsd")
		// 	totalCost = cost + estimatedPromptCost

		// 	c.Set("costInUsd", totalCost)
		// 	c.Set("completionTokenCount", tks+anthropicCompletionMagicNum)
		// }()

		telemetry.Incr("bricksllm.proxy.get_completion_handler.streaming_requests", nil, 1)

		eventName := ""
		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					telemetry.Incr("bricksllm.proxy.get_completion_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from anthropic completion response", prod, err)

					return false
				}

				telemetry.Incr("bricksllm.proxy.get_completion_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from anthropic streaming response", prod, err)

				apiErr := &anthropic.ErrorResponse{
					Error: &anthropic.Error{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_completion_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for anthropic streaming error response", prod, err)
					return false
				}

				c.SSEvent(" error", string(bytes))

				messageStop := &anthropic.MessagesStreamMessageStop{
					Type: "message_stop",
				}

				bytes, err = json.Marshal(messageStop)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for anthropic streaming message stop", prod, err)
					return false
				}

				c.SSEvent(" message_stop", string(bytes))
				return false
			}

			streamingResponse = append(streamingResponse, raw)

			noSpaceLine := bytes.TrimSpace(raw)
			if len(noSpaceLine) == 0 {
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventCompletionPrefix) {
				eventName = " completion"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventPingPrefix) {
				eventName = " ping"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventErrorPrefix) {
				eventName = " error"
				return true
			}

			noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
			c.SSEvent(eventName, " "+string(noPrefixLine))

			chatCompletionResp := &anthropic.CompletionResponse{}
			err = json.Unmarshal(noPrefixLine, chatCompletionResp)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_completion_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling anthropic completion stream response", prod, err)
			}

			if err == nil {
				content += chatCompletionResp.Completion
			}

			return true
		})

		telemetry.Timing("bricksllm.proxy.get_completion_handler.streaming_latency", time.Since(start), nil, 1)
	}
}

var (
	eventMessageStart      = []byte("event: message_start")
	eventMessageDelta      = []byte("event: message_delta")
	eventMessageStop       = []byte("event: message_stop")
	eventContentBlockStart = []byte("event: content_block_start")
	eventContentBlockDelta = []byte("event: content_block_delta")
	eventContentBlockStop  = []byte("event: content_block_stop")
)

func getMessagesHandler(prod, private bool, client http.Client, e anthropicEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_messages_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", c.Request.Body)
		if err != nil {
			logError(log, "error when creating anthropic http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create anthropic http request")
			return
		}

		raw, exists := c.Get("key")
		_, ok := raw.(*key.ResponseKey)
		if !exists || !ok {
			telemetry.Incr("bricksllm.proxy.get_messages_handler.api_key_not_registered", nil, 1)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
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
			telemetry.Incr("bricksllm.proxy.get_messages_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to anthropic", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to anthropic")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		model := c.GetString("model")

		if !isStreaming && res.StatusCode == http.StatusOK {
			dur := time.Now().Sub(start)
			telemetry.Timing("bricksllm.proxy.get_messages_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading anthropic http messages response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read anthropic response body")
				return
			}

			var cost float64 = 0
			completionTokens := 0
			promptTokens := 0

			completionRes := &anthropic.MessagesResponse{}
			telemetry.Incr("bricksllm.proxy.get_messages_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_messages_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, completionRes)
			if err != nil {
				logError(log, "error when unmarshalling anthropic http completion response body", prod, err)
			}

			if err == nil {
				logCompletionResponse(log, bytes, prod, private)
				completionTokens = completionRes.Usage.OutputTokens
				promptTokens = completionRes.Usage.InputTokens
				cost, err = e.EstimateTotalCost(model, promptTokens, completionTokens)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating anthropic cost", prod, err)
				}
			}

			c.Set("costInUsd", cost)
			c.Set("promptTokenCount", promptTokens)
			c.Set("completionTokenCount", completionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_messages_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_messages_handler.error_response", nil, 1)
			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading anthropic http messages response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read anthropic response body")
				return
			}

			logAnthropicErrorResponse(log, bytes, prod)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		var totalCost float64 = 0

		streamingResponse := [][]byte{}
		defer func() {
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		response := &anthropic.MessagesResponse{}
		defer func() {
			tks := response.Usage.OutputTokens
			model := c.GetString("model")
			cost, err := e.EstimateCompletionCost(model, tks)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_messages_handler.estimate_messages_cost_error", nil, 1)
				logError(log, "error when estimating anthropic messages stream cost", prod, err)
			}

			estimatedPromptCost, err := e.EstimatePromptCost(model, response.Usage.InputTokens)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_messages_handler.estimate_prompt_cost_error", nil, 1)
				logError(log, "error when estimating anthropic prompt cost", prod, err)
			}

			totalCost = cost + estimatedPromptCost

			c.Set("costInUsd", totalCost)
			c.Set("promptTokenCount", response.Usage.InputTokens)
			c.Set("completionTokenCount", tks)
		}()

		telemetry.Incr("bricksllm.proxy.get_messages_handler.streaming_requests", nil, 1)

		eventName := ""
		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				telemetry.Incr("bricksllm.proxy.get_messages_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from anthropic streaming response", prod, err)

				apiErr := &anthropic.ErrorResponse{
					Error: &anthropic.Error{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for anthropic streaming error response", prod, err)
					return false
				}

				c.SSEvent(" error", string(bytes))

				messageStop := &anthropic.MessagesStreamMessageStop{
					Type: "message_stop",
				}

				bytes, err = json.Marshal(messageStop)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for anthropic streaming message stop", prod, err)
					return false
				}

				c.SSEvent(" message_stop", string(bytes))
				return false
			}

			streamingResponse = append(streamingResponse, raw)

			noSpaceLine := bytes.TrimSpace(raw)
			if len(noSpaceLine) == 0 {
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventMessageStart) {
				eventName = " message_start"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventMessageDelta) {
				eventName = " message_delta"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventMessageStop) {
				eventName = " message_stop"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventContentBlockStart) {
				eventName = " content_block_start"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventContentBlockDelta) {
				eventName = " content_block_delta"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventContentBlockStop) {
				eventName = " content_block_stop"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventPingPrefix) {
				eventName = " ping"
				return true
			}

			if bytes.HasPrefix(noSpaceLine, eventErrorPrefix) {
				eventName = " error"
				return true
			}

			noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
			c.SSEvent(eventName, " "+string(noPrefixLine))

			if eventName == " message_start" {
				messageStart := &anthropic.MessagesStreamMessageStart{}
				err = json.Unmarshal(noPrefixLine, messageStart)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.message_start_response_unmarshall_error", nil, 1)
					logError(log, "error when unmarshalling anthropic message stream response message_start", prod, err)
					return true
				}

				response.Usage.InputTokens = messageStart.Message.Usage.InputTokens
			}

			if eventName == " message_delta" {
				messageDelta := &anthropic.MessagesStreamMessageDelta{}

				err = json.Unmarshal(noPrefixLine, messageDelta)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.message_delta_response_unmarshall_error", nil, 1)
					logError(log, "error when unmarshalling anthropic message stream response message_delta", prod, err)
					return true
				}

				response.Usage.OutputTokens = messageDelta.Usage.OutputTokens
			}

			if eventName == " content_block_start" {
				contentBlockStart := &anthropic.MessagesStreamBlockStart{}
				err = json.Unmarshal(noPrefixLine, contentBlockStart)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.content_block_start_response_unmarshall_error", nil, 1)
					logError(log, "error when unmarshalling anthropic message stream response content_block_start", prod, err)
					return true
				}
			}

			if eventName == " content_block_delta" {
				contentBlockDelta := &anthropic.MessagesStreamBlockDelta{}
				err = json.Unmarshal(noPrefixLine, contentBlockDelta)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_messages_handler.content_block_delta_response_unmarshall_error", nil, 1)
					logError(log, "error when unmarshalling anthropic message stream response content_block_delta", prod, err)
					return true
				}
			}

			return true
		})

		telemetry.Timing("bricksllm.proxy.get_messages_handler.streaming_latency", time.Since(start), nil, 1)
	}
}

func logAnthropicErrorResponse(log *zap.Logger, data []byte, prod bool) {
	cr := &anthropic.ErrorResponse{}
	err := json.Unmarshal(data, cr)
	if err != nil {
		logError(log, "error when unmarshalling anthropic error response", prod, err)
		return
	}

	if prod {
		fields := []zapcore.Field{}

		if cr.Error != nil {
			fields = append(fields, zap.Any("error", cr.Error))
		}

		log.Info("anthropic error response", fields...)
	}
}
