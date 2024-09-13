package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/bricks-cloud/bricksllm/internal/telemetry"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func setAnthropicVersionIfExists(version string, req *anthropic.BedrockMessageRequest) {
	if req != nil && len(version) > 0 {
		req.AnthropicVersion = version
	}
}

func getBedrockCompletionHandler(prod bool, e anthropicEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.read_all_error", nil, 1)
			log.Error("error when reading claude req data from body", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read claude req data from body")
			return
		}

		anthropicReq := &anthropic.CompletionRequest{}
		err = json.Unmarshal(body, anthropicReq)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.unmarshal_anthropic_completion_request_error", nil, 1)
			log.Error("error when unmarshalling anthropic completion request", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to unmarshal anthropic completion request")
			return
		}

		req := &anthropic.BedrockCompletionRequest{}
		err = json.Unmarshal(body, req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.unmarshal_bedrock_completion_request_error", nil, 1)
			log.Error("error when unmarshalling bedrock completion request", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to unmarshal bedrock completion request")
			return
		}

		bs, err := json.Marshal(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.marshal_bedrock_completion_request_error", nil, 1)
			log.Error("error when marshalling bedrock completion request", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to marshal bedrock completion request")
			return
		}

		keyId := c.GetString("awsAccessKeyId")
		secretKey := c.GetString("awsSecretAccessKey")
		region := c.GetString("awsRegion")

		if len(keyId) == 0 || len(secretKey) == 0 || len(region) == 0 {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.auth_error", nil, 1)
			log.Error("key id, secret key or region is missing", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] auth credentials are missing")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID: keyId, SecretAccessKey: secretKey,
					Source: "BricksLLM Credentials",
				},
			}),
			config.WithRegion(region))

		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.aws_config_creation_error", nil, 1)
			log.Error("error when creating aws config", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create aws config")
			return
		}

		client := bedrockruntime.NewFromConfig(cfg)
		stream := c.GetBool("stream")

		ctx, cancel = context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		start := time.Now()

		if !stream {
			output, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
				ModelId:     &anthropicReq.Model,
				ContentType: aws.String("application/json"),
				Body:        bs,
			})

			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.error_response", nil, 1)
				telemetry.Timing("bricksllm.proxy.get_bedrock_completion_handler.error_latency", time.Since(start), nil, 1)

				log.Error("error when invoking bedrock model", []zapcore.Field{zap.Error(err)}...)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to invoke bedrock model")
				return
			}

			completionRes := &anthropic.BedrockCompletionResponse{}
			err = json.Unmarshal(output.Body, completionRes)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.unmarshal_bedrock_completion_response_error", nil, 1)
				logError(log, "error when unmarshalling bedrock anthropic completion response body", prod, err)
			}

			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_bedrock_completion_handler.success_latency", time.Since(start), nil, 1)

			c.Set("content", completionRes.Completion)

			c.Data(http.StatusOK, "application/json", output.Body)
			return
		}

		telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.streaming_requests", nil, 1)

		streamOutput, err := client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
			ModelId:     &anthropicReq.Model,
			ContentType: aws.String("application/json"),
			Body:        bs,
		})

		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.invoking_model_with_streaming_response_error", nil, 1)
			log.Error("error when invoking bedrock model with streaming responses", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to invoke bedrock model with stream response")
			return
		}

		streamingResponse := [][]byte{}
		promptTokenCount := 0
		completionTokenCount := 0

		defer func() {
			model := c.GetString("model")
			translatedModel := util.TranslateBedrockModelToAnthropicModel(model)
			compeltionCost, err := e.EstimateCompletionCost(translatedModel, completionTokenCount)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.estimate_completion_cost_error", nil, 1)
				logError(log, "error when estimating bedrock completion cost", prod, err)
			}

			promptCost, err := e.EstimatePromptCost(translatedModel, promptTokenCount)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.estimate_prompt_cost_error", nil, 1)
				logError(log, "error when estimating bedrock prompt cost", prod, err)
			}

			c.Set("costInUsd", compeltionCost+promptCost)
			c.Set("promptTokenCount", promptTokenCount)
			c.Set("completionTokenCount", completionTokenCount)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		eventName := ""
		c.Stream(func(w io.Writer) bool {
			for event := range streamOutput.GetStream().Events() {
				switch v := event.(type) {
				case *types.ResponseStreamMemberChunk:
					raw := v.Value.Bytes
					noSpaceLine := bytes.TrimSpace(raw)
					if len(noSpaceLine) == 0 {
						return true
					}

					eventName = getEventNameFromLine(noSpaceLine)
					if len(eventName) == 0 {
						return true
					}

					chatCompletionResp := &anthropic.BedrockCompletionResponse{}
					if eventName == " completion" {
						err := json.NewDecoder(bytes.NewReader(noSpaceLine)).Decode(&chatCompletionResp)
						if err != nil {
							telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.bedrock_completion_stream_response_unmarshall_error", nil, 1)
							log.Error("error when unmarshalling bedrock streaming response chunks", []zapcore.Field{zap.Error(err)}...)
							return false
						}

						if chatCompletionResp.Metrics != nil {
							promptTokenCount = chatCompletionResp.Metrics.InputTokenCount
							completionTokenCount = chatCompletionResp.Metrics.OutputTokenCount
						}
					}

					noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
					c.SSEvent(eventName, " "+string(noPrefixLine))

					streamingResponse = append(streamingResponse, raw)
					if len(chatCompletionResp.StopReason) != 0 {
						return false
					}
				default:
					telemetry.Incr("bricksllm.proxy.get_bedrock_completion_handler.bedrock_completion_stream_response_unkown_error", nil, 1)
					return false
				}
			}

			telemetry.Timing("bricksllm.proxy.get_bedrock_completion_handler.streaming_latency", time.Since(start), nil, 1)
			return false
		})
	}
}

var (
	bedrockEventMessageStart      = []byte(`{"type":"message_start"`)
	bedrockEventMessageDelta      = []byte(`{"type":"message_delta"`)
	bedrockEventMessageStop       = []byte(`{"type":"message_stop"`)
	bedrockEventContentBlockStart = []byte(`{"type":"content_block_start"`)
	bedrockEventContentBlockDelta = []byte(`{"type":"content_block_delta"`)
	bedrockEventContentBlockStop  = []byte(`{"type":"content_block_stop"`)
	bedrockEventPing              = []byte(`{"type":"ping"`)
	bedrockEventError             = []byte(`{"type":"error"`)
	bedrockEventCompletion        = []byte(`{"type":"completion"`)
)

func getEventNameFromLine(line []byte) string {
	if bytes.HasPrefix(line, bedrockEventMessageStart) {
		return " message_start"
	}

	if bytes.HasPrefix(line, bedrockEventMessageDelta) {
		return " message_delta"
	}

	if bytes.HasPrefix(line, bedrockEventMessageStop) {
		return " message_stop"
	}

	if bytes.HasPrefix(line, bedrockEventContentBlockStart) {
		return " content_block_start"
	}

	if bytes.HasPrefix(line, bedrockEventContentBlockDelta) {
		return " content_block_delta"
	}

	if bytes.HasPrefix(line, bedrockEventContentBlockStop) {
		return " content_block_stop"
	}

	if bytes.HasPrefix(line, bedrockEventPing) {
		return " ping"
	}

	if bytes.HasPrefix(line, bedrockEventError) {
		return " error"
	}

	if bytes.HasPrefix(line, bedrockEventCompletion) {
		return " completion"
	}

	return ""
}

func getBedrockMessagesHandler(prod bool, e anthropicEstimator, timeOut time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.GetLogFromCtx(c)
		telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.requests", nil, 1)

		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.read_all_error", nil, 1)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read claude req data from body")
			return
		}

		anthropicReq := &anthropic.MessagesRequest{}
		err = json.Unmarshal(body, anthropicReq)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.unmarshal_anthropic_messages_request_error", nil, 1)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to unmarshal anthropic messages request")
			return
		}

		req := &anthropic.BedrockMessageRequest{}
		err = json.Unmarshal(body, req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.unmarshal_bedrock_messages_request_error", nil, 1)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to unmarshal bedrock messages request")
			return
		}

		setAnthropicVersionIfExists(c.GetHeader("anthropic-version"), req)

		bs, err := json.Marshal(req)
		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.marshal_error", nil, 1)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to marshal bedrock messages request")
			return
		}

		keyId := c.GetString("awsAccessKeyId")
		secretKey := c.GetString("awsSecretAccessKey")
		region := c.GetString("awsRegion")

		if len(keyId) == 0 || len(secretKey) == 0 || len(region) == 0 {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.auth_error", nil, 1)
			log.Error("key id, secret key or region is missing", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusUnauthorized, "[BricksLLM] auth credentials are missing")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID: keyId, SecretAccessKey: secretKey,
					Source: "BricksLLM Credentials",
				},
			}),
			config.WithRegion(region))

		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.aws_config_creation_error", nil, 1)
			log.Error("error when creating aws config", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create aws config")
			return
		}

		client := bedrockruntime.NewFromConfig(cfg)
		stream := c.GetBool("stream")

		ctx, cancel = context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		start := time.Now()

		if !stream {
			output, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
				ModelId:     &anthropicReq.Model,
				ContentType: aws.String("application/json"),
				Body:        bs,
			})

			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.error_response", nil, 1)
				telemetry.Timing("bricksllm.proxy.get_bedrock_messages_handler.error_latency", time.Since(start), nil, 1)

				log.Error("error when invoking bedrock model", []zapcore.Field{zap.Error(err)}...)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to invoke bedrock model")
				return
			}

			var cost float64 = 0
			completionTokens := 0
			promptTokens := 0

			messagesRes := &anthropic.MessagesResponse{}
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.success", nil, 1)
			telemetry.Timing("bricksllm.proxy.get_bedrock_messages_handler.success_latency", time.Since(start), nil, 1)

			err = json.Unmarshal(output.Body, messagesRes)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.unmarshal_bedrock_messages_response_error", nil, 1)
				logError(log, "error when unmarshalling bedrock messages response body", prod, err)
			}

			if err == nil {
				completionTokens = messagesRes.Usage.OutputTokens
				promptTokens = messagesRes.Usage.InputTokens

				model := c.GetString("model")
				translated := util.TranslateBedrockModelToAnthropicModel(model)

				cost, err = e.EstimateTotalCost(translated, promptTokens, completionTokens)
				if err != nil {
					telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.estimate_total_cost_error", nil, 1)
					logError(log, "error when estimating anthropic cost", prod, err)
				}
			}

			c.Set("costInUsd", cost)
			c.Set("promptTokenCount", promptTokens)
			c.Set("completionTokenCount", completionTokens)

			c.Data(http.StatusOK, "application/json", output.Body)
			return
		}

		telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.streaming_requests", nil, 1)

		streamOutput, err := client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
			ModelId:     &anthropicReq.Model,
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
			Body:        bs,
		})

		if err != nil {
			telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.invoking_model_with_streaming_response_error", nil, 1)

			log.Error("error when invoking bedrock model with streaming responses", []zapcore.Field{zap.Error(err)}...)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to invoke model request with stream response")
			return
		}

		streamingResponse := [][]byte{}
		promptTokenCount := 0
		completionTokenCount := 0

		defer func() {
			model := c.GetString("model")
			translatedModel := util.TranslateBedrockModelToAnthropicModel(model)
			compeltionCost, err := e.EstimateCompletionCost(translatedModel, completionTokenCount)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.estimate_completion_cost_error", nil, 1)
				logError(log, "error when estimating bedrock completion cost", prod, err)
			}

			promptCost, err := e.EstimatePromptCost(translatedModel, promptTokenCount)
			if err != nil {
				telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.estimate_prompt_cost_error", nil, 1)
				logError(log, "error when estimating bedrock prompt cost", prod, err)
			}

			c.Set("costInUsd", compeltionCost+promptCost)
			c.Set("promptTokenCount", promptTokenCount)
			c.Set("completionTokenCount", completionTokenCount)
			c.Set("streaming_response", bytes.Join(streamingResponse, []byte{'\n'}))
		}()

		eventName := ""
		c.Stream(func(w io.Writer) bool {
			content := ""
			for event := range streamOutput.GetStream().Events() {
				switch v := event.(type) {
				case *types.ResponseStreamMemberChunk:
					raw := v.Value.Bytes
					streamingResponse = append(streamingResponse, raw)

					noSpaceLine := bytes.TrimSpace(raw)
					if len(noSpaceLine) == 0 {
						return true
					}

					eventName = getEventNameFromLine(noSpaceLine)
					if len(eventName) == 0 {
						return true
					}

					if eventName == " message_stop" {
						stopResp := &anthropic.BedrockMessagesStopResponse{}
						err := json.NewDecoder(bytes.NewReader(raw)).Decode(&stopResp)
						if err != nil {
							telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.bedrock_messages_stop_response_unmarshall_error", nil, 1)
							log.Error("error when unmarshalling bedrock messages stop response response chunks", []zapcore.Field{zap.Error(err)}...)

							return false
						}

						if stopResp.Metrics != nil {
							promptTokenCount = stopResp.Metrics.InputTokenCount
							completionTokenCount = stopResp.Metrics.OutputTokenCount
						}
					}

					if eventName == " content_block_delta" {
						chatCompletionResp := &anthropic.MessagesStreamBlockDelta{}
						err := json.NewDecoder(bytes.NewReader(raw)).Decode(&chatCompletionResp)
						if err != nil {
							telemetry.Incr("bricksllm.proxy.get_bedrock_messages_handler.bedrock_messages_content_block_response_unmarshall_error", nil, 1)
							log.Error("error when unmarshalling bedrock messages content block response chunks", []zapcore.Field{zap.Error(err)}...)

							return false
						}

						content += chatCompletionResp.Delta.Text
					}

					c.SSEvent(eventName, " "+string(noSpaceLine))

					if eventName == " message_stop" {
						return false
					}
				default:
					telemetry.Timing("bricksllm.proxy.get_bedrock_messages_handler.streaming_latency", time.Since(start), nil, 1)
					return false
				}
			}

			telemetry.Timing("bricksllm.proxy.get_bedrock_messages_handler.streaming_latency", time.Since(start), nil, 1)
			return false
		})
	}
}
