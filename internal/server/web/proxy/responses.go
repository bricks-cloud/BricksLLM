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

	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	responsesOpenai "github.com/openai/openai-go/v3/responses"
	goopenai "github.com/sashabaranov/go-openai"
)

func getResponsesHandler(prod, private bool, client http.Client, e estimator) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_responses_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), c.GetDuration("requestTimeout"))
		defer cancel()

		wildcard := c.Param("wildcard")
		url := fmt.Sprintf("https://api.openai.com/v1/responses%s", wildcard)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, c.Request.Body)
		if err != nil {
			logError(log, "error when creating openai http request", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
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
			telemetry.Incr("bricksllm.proxy.get_responses_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to openai", prod, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}

		defer res.Body.Close()

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		model := c.GetString("model")

		if res.StatusCode == http.StatusOK && !isStreaming {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_responses_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http response api response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			var cost float64 = 0
			resp := &responsesOpenai.Response{}
			telemetry.Incr("bricksllm.proxy.get_responses_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_responses_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, resp)
			if err != nil {
				logError(log, "error when unmarshalling openai http response api response body", prod, err)
			}

			if err == nil {
				logResponsesResponse(log, prod, private, resp)
				cost, err = e.EstimateResponseApiTotalCost(model, resp.Usage)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating openai cost", prod, err)
				}
				reqResp, _ := ginCtxGetResponsesRequest(c)
				containerCost, err := e.EstimateResponseApiToolCreateContainerCost(reqResp)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_tool_container_cost_error", nil, 1)
					logError(log, "error when estimating openai tool container cost", prod, err)
				}
				cost += containerCost
				toolsCost, err := e.EstimateResponseApiToolCallsCost(resp.Tools, model)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_tool_calls_cost_error", nil, 1)
					logError(log, "error when estimating openai tool calls cost", prod, err)
				}
				cost += toolsCost
			}

			c.Set("costInUsd", cost)

			promptTokenCount, err := int64ToInt(resp.Usage.InputTokens)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_responses_handler.int64_to_int_error", nil, 1)
				logError(log, "error when converting int64 to int for prompt token count", prod, err)
			}

			completionTokenCount, err := int64ToInt(resp.Usage.OutputTokens)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_responses_handler.int64_to_int_error", nil, 1)
				logError(log, "error when converting int64 to int for completion token count", prod, err)
			}

			c.Set("promptTokenCount", promptTokenCount)
			c.Set("completionTokenCount", completionTokenCount)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			telemetry.Timing("bricksllm.proxy.get_responses_handler.error_latency", dur, nil, 1)
			telemetry.Incr("bricksllm.proxy.get_responses_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading openai http response api response body", prod, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
				return
			}

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai response api error response body", prod, err)
			}

			logOpenAiError(log, prod, errorRes)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		content := ""
		streamingResponse := [][]byte{}

		streamCost := 0.0
		streamPromptTokenCount := 0
		streamCompletionTokenCount := 0

		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))

			c.Set("costInUsd", streamCost)
			c.Set("promptTokenCount", streamPromptTokenCount)
			c.Set("completionTokenCount", streamCompletionTokenCount)
		}()

		telemetry.Incr("bricksllm.proxy.get_responses_handler.streaming_requests", nil, 1)
		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					telemetry.Incr("bricksllm.proxy.get_responses_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from openai responses api response", prod, err)

					return false
				}

				telemetry.Incr("bricksllm.proxy.get_responses_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from openai responses api response", prod, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				errBytes, err := json.Marshal(apiErr)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_responses_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for openai streaming responses api error response", prod, err)
					return false
				}

				c.SSEvent("", string(errBytes))
				c.SSEvent("", " [DONE]")
				return false
			}

			streamingResponse = append(streamingResponse, raw)

			noSpaceLine := bytes.TrimSpace(raw)
			if !bytes.HasPrefix(noSpaceLine, headerData) {
				return true
			}

			noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
			c.SSEvent("", " "+string(noPrefixLine))

			if string(noPrefixLine) == "[DONE]" {
				return false
			}

			responsesStreamResp := &responsesOpenai.ResponseStreamEventUnion{}
			err = json.Unmarshal(noPrefixLine, responsesStreamResp)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_responses_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling openai responses api stream response", prod, err)
			}
			if err == nil {
				textDelta := responsesStreamResp.AsResponseOutputTextDelta().Delta
				if len(textDelta) > 0 {
					content += textDelta
				}

				if responsesStreamResp.Response.Status == "completed" {
					streamCost, err = e.EstimateResponseApiTotalCost(model, responsesStreamResp.Response.Usage)
					if err != nil {
						telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_total_cost_error", nil, 1)
						logError(log, "error when estimating openai cost", prod, err)
					}
					reqResp, _ := ginCtxGetResponsesRequest(c)
					containerCost, err := e.EstimateResponseApiToolCreateContainerCost(reqResp)
					if err != nil {
						telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_tool_container_cost_error", nil, 1)
						logError(log, "error when estimating openai tool container cost", prod, err)
					}
					streamCost += containerCost
					toolsCost, err := e.EstimateResponseApiToolCallsCost(responsesStreamResp.Response.Tools, model)
					if err != nil {
						telemetry.Incr("bricksllm.proxy.get_chat_completion_handler.estimate_tool_calls_cost_error", nil, 1)
						logError(log, "error when estimating openai tool calls cost", prod, err)
					}
					streamCost += toolsCost
					streamPromptTokenCount, err = int64ToInt(responsesStreamResp.Response.Usage.InputTokens)
					if err != nil {
						telemetry.Incr("bricksllm.proxy.get_responses_handler.int64_to_int_error", nil, 1)
						logError(log, "error when converting int64 to int for prompt token count", prod, err)
					}

					streamCompletionTokenCount, err = int64ToInt(responsesStreamResp.Response.Usage.OutputTokens)
					if err != nil {
						telemetry.Incr("bricksllm.proxy.get_responses_handler.int64_to_int_error", nil, 1)
						logError(log, "error when converting int64 to int for completion token count", prod, err)
					}
				}
			}
			return true
		})
		telemetry.Timing("bricksllm.proxy.get_chat_completion_handler.streaming_latency", time.Since(start), nil, 1)
	}
}

func int64ToInt(src int64) (int, error) {
	if src > int64(int(^uint(0)>>1)) {
		return 0, fmt.Errorf("int64 value %d overflows int", src)
	}
	return int(src), nil
}

func ginCtxSetResponsesRequest(c *gin.Context, req *openai.ResponseRequest) {
	c.Set("responses_request", req)
}

func ginCtxGetResponsesRequest(c *gin.Context) (*openai.ResponseRequest, error) {
	reqAny, exists := c.Get("responses_request")
	if !exists {
		return nil, errors.New("responses request not found in gin context")
	}

	req, ok := reqAny.(*openai.ResponseRequest)
	if !ok {
		return nil, errors.New("responses request in gin context has invalid type")
	}

	return req, nil
}
