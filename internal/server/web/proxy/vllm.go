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

	"github.com/bricks-cloud/bricksllm/internal/provider/vllm"
	"github.com/bricks-cloud/bricksllm/internal/stats"
	"github.com/gin-gonic/gin"
	goopenai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getVllmCompletionsHandler(prod, private bool, client http.Client, e estimator, log *zap.Logger, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.proxy.get_vllm_completions_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(correlationId)
		url := c.GetString("vllmUrl")
		if len(url) == 0 {
			logError(log, "vllm url cannot be empty", prod, cid, errors.New("url is empty"))
			JSON(c, http.StatusInternalServerError, "[BricksLLM] vllm url is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/completions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating vllm http request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create vllm http request")
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
			stats.Incr("bricksllm.proxy.get_vllm_completions_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to vllm", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to vllm")
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
			stats.Timing("bricksllm.proxy.get_vllm_completions_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading vllm http completions response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read vllm response body")
				return
			}

			cr := &goopenai.CompletionResponse{}
			stats.Incr("bricksllm.proxy.get_vllm_completions_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_vllm_completions_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, cr)
			if err != nil {
				logError(log, "error when unmarshalling openai http chat completion response body", prod, cid, err)
			}

			if err == nil {
				logVllmCompletionResponse(log, cr, prod, private, cid)
			}

			c.Set("promptTokenCount", cr.Usage.PromptTokens)
			c.Set("completionTokenCount", cr.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_vllm_completions_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_vllm_completions_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading vllm http chat completion response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read vllm response body")
				return
			}

			errorRes := &goopenai.ErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai chat completion error response body", prod, cid, err)
			}

			logOpenAiError(log, prod, cid, errorRes)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		content := ""
		streamingResponse := [][]byte{}
		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		stats.Incr("bricksllm.proxy.get_vllm_completions_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_vllm_completions_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from vllm completions response", prod, cid, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_vllm_completions_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from vllm completions response", prod, cid, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_vllm_completions_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for vllm streaming chat completion error response", prod, cid, err)
					return false
				}

				c.SSEvent("", string(bytes))
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

			completionsStreamResp := &goopenai.CompletionResponse{}
			err = json.Unmarshal(noPrefixLine, completionsStreamResp)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_vllm_completions_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling vllm completions stream response", prod, cid, err)
			}

			if err == nil {
				if len(completionsStreamResp.Choices) > 0 && len(completionsStreamResp.Choices[0].Text) != 0 {
					content += completionsStreamResp.Choices[0].Text
				}
			}

			return true
		})

		stats.Timing("bricksllm.proxy.get_vllm_completions_handler.streaming_latency", time.Since(start), nil, 1)

	}
}

func logVllmCompletionRequest(log *zap.Logger, cr *vllm.CompletionRequest, prod, private bool, cid string) {

	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", cr.Model),
			zap.String("suffix", cr.Suffix),
			zap.Int("max_tokens", cr.MaxTokens),
			zap.Float32("temperature", cr.Temperature),
			zap.Float32("top_p", cr.TopP),
			zap.Int("n", cr.N),
			zap.Any("stop", cr.Stop),
			zap.Bool("echo", cr.Echo),
			zap.Int("best_of", cr.BestOf),
			zap.Float32("presence_penalty", cr.PresencePenalty),
			zap.Any("response_format", cr.Stream),
			zap.Any("logit_bias", cr.LogitBias),
			zap.Bool("use_beam_search", cr.UseBeamSearch),
			zap.Int("top_k", cr.TopK),
			zap.Int("min_p", cr.MinP),
			zap.Float64("repetition_penalty", cr.RepetitionPenalty),
			zap.Float64("length_penalty", cr.LengthPenalty),
			zap.Bool("early_stopping", cr.EarlyStopping),
			zap.Any("stop_token_ids", cr.StopTokenIds),
			zap.Bool("ignore_eos", cr.IgnoreEos),
			zap.Int("min_tokens", cr.MinTokens),
			zap.Bool("skip_special_tokens", cr.SkipSpecialTokens),
			zap.Bool("spaces_between_special_tokens", cr.SpacesBetweenSpecialTokens),
		}

		if !private {
			fields = append(fields, zap.String("user", cr.User))
			fields = append(fields, zap.Any("prompt", cr.Prompt))
		}

		log.Info("vllm completion request", fields...)
	}
}

func logVllmChatCompletionRequest(log *zap.Logger, cr *vllm.ChatRequest, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("model", cr.Model),
			zap.Int("max_tokens", cr.MaxTokens),
			zap.Float32("temperature", cr.Temperature),
			zap.Float32("top_p", cr.TopP),
			zap.Int("n", cr.N),
			zap.Any("stop", cr.Stop),
			zap.Float32("presence_penalty", cr.PresencePenalty),
			zap.Any("response_format", cr.Stream),
			zap.Intp("seed", cr.Seed),
			zap.Any("logit_bias", cr.LogitBias),
			zap.Bool("logit_probs", cr.LogProbs),
			zap.Int("top_log_probs", cr.TopLogProbs),
			zap.Int("best_of", cr.BestOf),
			zap.Bool("use_beam_search", cr.UseBeamSearch),
			zap.Int("top_k", cr.TopK),
			zap.Int("min_p", cr.MinP),
			zap.Float64("repetition_penalty", cr.RepetitionPenalty),
			zap.Float64("length_penalty", cr.LengthPenalty),
			zap.Bool("early_stopping", cr.EarlyStopping),
			zap.Any("stop_token_ids", cr.StopTokenIds),
			zap.Bool("ignore_eos", cr.IgnoreEos),
			zap.Int("min_tokens", cr.MinTokens),
			zap.Bool("skip_special_tokens", cr.SkipSpecialTokens),
			zap.Bool("spaces_between_special_tokens", cr.SpacesBetweenSpecialTokens),
		}

		if !private {
			fields = append(fields, zap.String("user", cr.User))
		}

		ccms := []goopenai.ChatCompletionMessage{}

		for _, m := range cr.Messages {
			toBeAdded := goopenai.ChatCompletionMessage{}

			if !private {
				toBeAdded.Content = m.Content
			}

			cmps := []goopenai.ChatMessagePart{}
			for _, p := range m.MultiContent {
				toBeAddedPart := goopenai.ChatMessagePart{
					Type: p.Type,
				}

				if !private {
					toBeAddedPart.ImageURL = p.ImageURL
					toBeAddedPart.Text = p.Text
				}

				cmps = append(cmps, toBeAddedPart)
			}

			m.MultiContent = cmps

			toBeAdded.Role = m.Role
			toBeAdded.Name = m.Name

			ccms = append(ccms, toBeAdded)
		}

		fields = append(fields, zap.Any("messages", ccms))

		log.Info("vllm completion request", fields...)
	}
}

func logVllmCompletionResponse(log *zap.Logger, cr *goopenai.CompletionResponse, prod, private bool, cid string) {
	if prod {
		fields := []zapcore.Field{
			zap.String(correlationId, cid),
			zap.String("id", cr.ID),
			zap.Int64("created", cr.Created),
			zap.String("model", cr.Model),
		}

		ccs := []goopenai.CompletionChoice{}

		for _, cc := range cr.Choices {
			toBeAdded := goopenai.CompletionChoice{}

			if !private {
				toBeAdded.Text = cc.Text
			}

			toBeAdded.FinishReason = cc.FinishReason
			toBeAdded.LogProbs = cc.LogProbs
			toBeAdded.Index = cc.Index

			ccs = append(ccs, toBeAdded)
		}

		fields = append(fields, zap.Any("choices", ccs))

		log.Info("vllm completion response", fields...)
	}
}

func getVllmChatCompletionsHandler(prod, private bool, client http.Client, e estimator, log *zap.Logger, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.requests", nil, 1)
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		cid := c.GetString(correlationId)
		url := c.GetString("vllmUrl")
		if len(url) == 0 {
			logError(log, "vllm url cannot be empty", prod, cid, errors.New("url is empty"))
			JSON(c, http.StatusInternalServerError, "[BricksLLM] vllm url is empty")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/chat/completions", c.Request.Body)
		if err != nil {
			logError(log, "error when creating vllm chat completions http request", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create vllm http request")
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
			stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.http_client_error", nil, 1)

			logError(log, "error when sending http request to vllm", prod, cid, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to vllm")
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
			stats.Timing("bricksllm.proxy.get_vllm_chat_completions_handler.latency", dur, nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading vllm chat completions response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read vllm response body")
				return
			}

			chatRes := &goopenai.ChatCompletionResponse{}
			stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.success", nil, 1)
			stats.Timing("bricksllm.proxy.get_vllm_chat_completions_handler.success_latency", dur, nil, 1)

			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling vllm chat completions response body", prod, cid, err)
			}

			if err == nil {
				logChatCompletionResponse(log, prod, private, cid, chatRes)
			}

			c.Set("promptTokenCount", chatRes.Usage.PromptTokens)
			c.Set("completionTokenCount", chatRes.Usage.CompletionTokens)

			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		if res.StatusCode != http.StatusOK {
			dur := time.Since(start)
			stats.Timing("bricksllm.proxy.get_vllm_chat_completions_handler.error_latency", dur, nil, 1)
			stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.error_response", nil, 1)

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				logError(log, "error when reading vllm chat completions response body", prod, cid, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read vllm response body")
				return
			}

			logAnthropicErrorResponse(log, bytes, prod, cid)
			c.Data(res.StatusCode, "application/json", bytes)
			return
		}

		buffer := bufio.NewReader(res.Body)
		content := ""
		streamingResponse := [][]byte{}
		defer func() {
			c.Set("content", content)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.streaming_requests", nil, 1)

		c.Stream(func(w io.Writer) bool {
			raw, err := buffer.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return false
				}

				if errors.Is(err, context.DeadlineExceeded) {
					stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.context_deadline_exceeded_error", nil, 1)
					logError(log, "context deadline exceeded when reading bytes from vllm chat completions response", prod, cid, err)

					return false
				}

				stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.read_bytes_error", nil, 1)
				logError(log, "error when reading bytes from vllm chat completions response", prod, cid, err)

				apiErr := &goopenai.ErrorResponse{
					Error: &goopenai.APIError{
						Type:    "bricksllm_error",
						Message: err.Error(),
					},
				}

				bytes, err := json.Marshal(apiErr)
				if err != nil {
					stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.json_marshal_error", nil, 1)
					logError(log, "error when marshalling bytes for streaming vllm chat completions error response", prod, cid, err)
					return false
				}

				c.SSEvent("", string(bytes))
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

			chatCompletionStreamResp := &goopenai.ChatCompletionStreamResponse{}
			err = json.Unmarshal(noPrefixLine, chatCompletionStreamResp)
			if err != nil {
				stats.Incr("bricksllm.proxy.get_vllm_chat_completions_handler.completion_response_unmarshall_error", nil, 1)
				logError(log, "error when unmarshalling vllm chat completions stream response", prod, cid, err)
			}

			if err == nil {
				if len(chatCompletionStreamResp.Choices) > 0 && len(chatCompletionStreamResp.Choices[0].Delta.Content) != 0 {
					content += chatCompletionStreamResp.Choices[0].Delta.Content
				}
			}

			return true
		})

		stats.Timing("bricksllm.proxy.get_vllm_chat_completions_handler.streaming_latency", time.Since(start), nil, 1)
	}
}
