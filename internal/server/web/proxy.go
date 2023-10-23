package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/bricks-cloud/bricksllm/internal/util"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	correlationId string = "correlationId"
)

type ProxyServer struct {
	server *http.Server
	log    *zap.Logger
}

type recorder interface {
	RecordKeySpend(keyId string, model string, micros int64, costLimitUnit key.TimeUnit) error
	RecordEvent(e *event.Event) error
}

func NewProxyServer(log *zap.Logger, mode, privacyMode string, m KeyManager, ks keyStorage, kms keyMemStorage, e estimator, v validator, r recorder, credential string, enc encrypter, rlm rateLimitManager) (*ProxyServer, error) {
	router := gin.New()
	prod := mode == "production"
	private := mode == "strict"

	router.Use(getProxyLogger(log, "proxy", prod))
	router.Use(getKeyValidator(kms, prod, private, e, v, ks, log, enc, rlm))

	client := http.Client{}

	router.POST("/api/providers/openai/v1/chat/completions", getChatCompletionHandler(r, prod, private, credential, client, kms, log, enc, e))

	srv := &http.Server{
		Addr:    ":8002",
		Handler: router,
	}

	return &ProxyServer{
		log:    log,
		server: srv,
	}, nil
}

func getChatCompletionHandler(r recorder, prod, private bool, credential string, client http.Client, kms keyMemStorage, log *zap.Logger, enc encrypter, e estimator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] context is empty")
			return
		}

		req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", c.Request.Body)
		id := c.GetString(correlationId)
		if err != nil {
			logError(log, "error when creating openai http request", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to create openai http request")
			return
		}

		split := strings.Split(c.Request.Header.Get("Authorization"), "Bearer ")
		if len(split) < 2 || len(split[1]) == 0 {
			JSON(c, http.StatusUnauthorized, "[BricksLLM] bearer token is not present")
			return
		}

		apiKey := split[1]
		hash := enc.Encrypt(apiKey)

		kc := kms.GetKey(hash)
		if kc == nil {
			JSON(c, http.StatusUnauthorized, "[BricksLLM] api key is not registered")
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", credential))

		res, err := client.Do(req)
		if err != nil {
			logError(log, "error when sending http request to openai", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}
		defer res.Body.Close()

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			logError(log, "error when reading openai http response body", prod, id, err)
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		var cost float64 = 0
		var model string = "unknown"

		if res.StatusCode == http.StatusOK {
			chatRes := &openai.ChatCompletionResponse{}
			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				logError(log, "error when unmarshalling openai http response body", prod, id, err)
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to parse openai response")
				return
			}

			logResponse(log, prod, private, id, chatRes)

			model = chatRes.Model

			cost, err := e.EstimateTotalCost(chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
			if err != nil {
				logError(log, "error when estimating openai cost", prod, id, err)
			}

			micros := int64(cost * 1000000)
			err = r.RecordKeySpend(kc.KeyId, chatRes.Model, micros, kc.CostLimitInUsdUnit)
			if err != nil {
				logError(log, "error when recording openai spend", prod, id, err)
			}
		}

		if res.StatusCode != http.StatusOK {
			errorRes := &openai.ChatCompletionErrorResponse{}
			err = json.Unmarshal(bytes, errorRes)
			if err != nil {
				logError(log, "error when unmarshalling openai http error response body", prod, id, err)
			}

			logOpenAiError(log, prod, id, errorRes)
		}

		evt := &event.Event{
			Id:        util.NewUuid(),
			CreatedAt: time.Now().Unix(),
			Tags:      kc.Tags,
			KeyId:     kc.KeyId,
			CostInUsd: cost,
			Provider:  provider.OpenAiProvider,
			Model:     model,
			Status:    res.StatusCode,
		}

		err = r.RecordEvent(evt)
		if err != nil {
			logError(log, "error when recording openai event", prod, id, err)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		c.Data(res.StatusCode, "application/json", bytes)
	}
}

func (ps *ProxyServer) Run() {
	go func() {
		ps.log.Info("proxy server listening at 8002")
		ps.log.Info("PORT 8002 | POST | /api/providers/openai/v1/chat/completions is ready for forwarding requests to openai")

		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ps.log.Sugar().Fatalf("error proxy server listening: %v", err)
			return
		}
	}()
}

func logResponse(log *zap.Logger, prod, private bool, cid string, r *openai.ChatCompletionResponse) {
	if prod {
		log.Info("openai response",
			zap.Time("createdAt", time.Now()),
			zap.String(correlationId, cid),
			zap.Object("response", zapcore.ObjectMarshalerFunc(
				func(enc zapcore.ObjectEncoder) error {
					enc.AddString("id", r.Id)
					enc.AddString("object", r.Object)
					enc.AddInt64("created", r.Created)
					enc.AddString("model", r.Model)
					enc.AddArray("choices", zapcore.ArrayMarshalerFunc(
						func(enc zapcore.ArrayEncoder) error {
							for _, c := range r.Choices {
								enc.AppendObject(zapcore.ObjectMarshalerFunc(
									func(enc zapcore.ObjectEncoder) error {
										enc.AddInt("index", c.Index)
										enc.AddObject("message", zapcore.ObjectMarshalerFunc(
											func(enc zapcore.ObjectEncoder) error {
												enc.AddString("role", c.Message.Role)
												if !private {
													enc.AddString("content", c.Message.Content)
												}
												return nil
											},
										))

										enc.AddString("finish_reason", c.FinishReason)
										return nil
									},
								))
							}
							return nil
						},
					))

					enc.AddObject("usage", zapcore.ObjectMarshalerFunc(
						func(enc zapcore.ObjectEncoder) error {
							enc.AddInt("prompt_tokens", r.Usage.PromptTokens)
							enc.AddInt("completion_tokens", r.Usage.CompletionTokens)
							enc.AddInt("total_tokens", r.Usage.TotalTokens)
							return nil
						},
					))
					return nil
				},
			)),
		)
	}
}

func logRequest(log *zap.Logger, prod, private bool, id string, r *openai.ChatCompletionRequest) {
	if prod {
		log.Info("openai request",
			zap.Time("createdAt", time.Now()),
			zap.String(correlationId, id),
			zap.Object("request", zapcore.ObjectMarshalerFunc(
				func(enc zapcore.ObjectEncoder) error {
					enc.AddString("model", r.Model)

					if len(r.Messages) != 0 {
						enc.AddArray("messages", zapcore.ArrayMarshalerFunc(
							func(enc zapcore.ArrayEncoder) error {
								for _, m := range r.Messages {
									err := enc.AppendObject(zapcore.ObjectMarshalerFunc(
										func(enc zapcore.ObjectEncoder) error {
											enc.AddString("name", m.Name)
											enc.AddString("role", m.Role)

											if m.FunctionCall != nil {
												enc.AddObject("function_call", zapcore.ObjectMarshalerFunc(
													func(enc zapcore.ObjectEncoder) error {
														enc.AddString("name", m.FunctionCall.Name)
														if !private {
															enc.AddString("arguments", m.FunctionCall.Arguments)
														}
														return nil
													},
												))
											}

											if !private {
												enc.AddString("content", m.Content)
											}

											return nil
										},
									))

									if err != nil {
										return err
									}
								}
								return nil
							},
						))
					}

					if len(r.Functions) != 0 {
						enc.AddArray("functions", zapcore.ArrayMarshalerFunc(
							func(enc zapcore.ArrayEncoder) error {
								for _, f := range r.Functions {
									err := enc.AppendObject(zapcore.ObjectMarshalerFunc(
										func(enc zapcore.ObjectEncoder) error {
											enc.AddString("name", f.Name)
											enc.AddString("description", f.Description)

											if f.Parameters != nil && !private {
												bs, err := json.Marshal(f.Parameters)
												if err != nil {
													return err
												}

												enc.AddString("parameters", string(bs))
											}

											return nil
										},
									))

									if err != nil {
										return err
									}

								}
								return nil
							},
						))
					}

					if r.MaxTokens != 0 {
						enc.AddInt("max_tokens", r.MaxTokens)
					}

					if r.Temperature != 0 {
						enc.AddFloat32("temperature", r.Temperature)
					}

					if r.TopP != 0 {
						enc.AddFloat32("top_p", r.TopP)
					}

					if r.N != 0 {
						enc.AddInt("n", r.N)
					}

					if r.Stream {
						enc.AddBool("stream", r.Stream)
					}

					if len(r.Stop) != 0 {
						enc.AddArray("stop", zapcore.ArrayMarshalerFunc(
							func(enc zapcore.ArrayEncoder) error {
								for _, s := range r.Stop {
									enc.AppendString(s)
								}
								return nil
							},
						))
					}

					if r.PresencePenalty != 0 {
						enc.AddFloat32("presence_penalty", r.PresencePenalty)
					}

					if r.FrequencyPenalty != 0 {
						enc.AddFloat32("frequency_penalty", r.FrequencyPenalty)
					}

					if len(r.LogitBias) != 0 {
						enc.AddObject("logit_bias", zapcore.ObjectMarshalerFunc(
							func(enc zapcore.ObjectEncoder) error {
								for k, v := range r.LogitBias {
									enc.AddInt(k, v)
								}
								return nil
							},
						))
					}

					if len(r.User) != 0 {
						enc.AddString("user", r.User)
					}

					return nil
				},
			)))
	}
}

func logOpenAiError(log *zap.Logger, prod bool, id string, errRes *openai.ChatCompletionErrorResponse) {
	if prod {
		log.Info("openai error response", zap.String(correlationId, id), zap.Any("error", errRes))
		return
	}

	log.Sugar().Infof("correlationId:%s | %s ", id, "openai error response")
}

func logError(log *zap.Logger, msg string, prod bool, id string, err error) {
	if prod {
		log.Debug(msg, zap.String(correlationId, id), zap.Error(err))
		return
	}

	log.Sugar().Debugf("correlationId:%s | %s | %v", id, msg, err)
}

func getProxyLogger(log *zap.Logger, prefix string, prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(correlationId, util.NewUuid())
		start := time.Now()
		c.Next()
		latency := time.Now().Sub(start).Milliseconds()
		if !prod {
			log.Sugar().Infof("%s | %d | %s | %s | %dms", prefix, c.Writer.Status(), c.Request.Method, c.FullPath(), latency)
		}

		if prod {
			log.Info("request to openai proxy",
				zap.String(correlationId, c.GetString(correlationId)),
				zap.String("keyId", c.GetString("keyId")),
				zap.Int("code", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", c.FullPath()),
				zap.Int64("lantecyInMs", latency),
			)
		}
	}
}

func (ps *ProxyServer) Shutdown(ctx context.Context) error {
	if err := ps.server.Shutdown(ctx); err != nil {
		ps.log.Sugar().Infof("error shutting down proxy server: %v", err)

		return err
	}

	return nil
}
