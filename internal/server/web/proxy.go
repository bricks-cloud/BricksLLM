package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bricks-cloud/bricksllm/internal/logger"
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/gin-gonic/gin"
)

type ProxyServer struct {
	server *http.Server
	logger logger.Logger
}

type recorder interface {
	RecordKeySpend(keyId string, model string, promptTks int, completionTks int) error
}

func NewProxyServer(log logger.Logger, m KeyManager, ks keyStorage, kms keyMemStorage, e estimator, v validator, r recorder, credential string) (*ProxyServer, error) {
	router := gin.New()
	router.Use(getKeyValidator(kms, e, v, ks, log))

	client := http.Client{}

	router.POST("/api/providers/openai", getOpenAiProxyHandler(r, credential, client, kms, log))

	srv := &http.Server{
		Addr:    ":8002",
		Handler: router,
	}

	log.Info("POST   /api/providers/openai is set up for forwarding requests to openai")

	return &ProxyServer{
		logger: log,
		server: srv,
	}, nil
}

func getOpenAiProxyHandler(r recorder, credential string, client http.Client, kms keyMemStorage, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", c.Request.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &openai.ChatCompletionErrorResponse{
				Error: &openai.ErrorContent{
					Message: "[BricksLLM] failed to create openai http request",
				},
			})
			return
		}

		split := strings.Split(c.Request.Header.Get("Authorization"), "Bearer ")
		if len(split) < 2 || len(split[1]) == 0 {
			c.JSON(http.StatusUnauthorized, &openai.ChatCompletionErrorResponse{
				Error: &openai.ErrorContent{
					Message: "[BricksLLM] bearer token is not present",
				},
			})
			return
		}

		apiKey := split[1]
		kc := kms.GetKey(apiKey)
		if kc == nil {
			c.JSON(http.StatusUnauthorized, &openai.ChatCompletionErrorResponse{
				Error: &openai.ErrorContent{
					Message: "[BricksLLM] api key is not registered",
				},
			})
			return
		}

		for name, values := range req.Header {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", credential))

		res, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &openai.ChatCompletionErrorResponse{
				Error: &openai.ErrorContent{
					Message: "[BricksLLM] failed to send http request to openai",
				},
			})
			return
		}
		defer res.Body.Close()

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &openai.ChatCompletionErrorResponse{
				Error: &openai.ErrorContent{
					Message: "[BricksLLM] failed to read openai response body",
				},
			})
			return
		}

		chatRes := &openai.ChatCompletionResponse{}
		err = json.Unmarshal(bytes, chatRes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &openai.ChatCompletionErrorResponse{
				Error: &openai.ErrorContent{
					Message: "[BricksLLM] failed to parse openai response",
				},
			})
			return
		}

		err = r.RecordKeySpend(kc.KeyId, chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
		if err != nil {
			log.Debugf("failed to record key spend for key: %s :%v", kc.KeyId, err)
		}

		for name, values := range res.Header {
			for _, value := range values {
				c.Header(name, value)
			}
		}

		io.Copy(c.Writer, res.Body)
		c.Status(res.StatusCode)
	}
}

func (ps *ProxyServer) Run() {
	go func() {
		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ps.logger.Fatalf("error proxy server listening: %v", err)
			return
		}
		ps.logger.Info("proxy server listening at 8002")
	}()

}

func (ps *ProxyServer) Shutdown(ctx context.Context) error {
	if err := ps.server.Shutdown(ctx); err != nil {
		ps.logger.Debugf("error shutting down proxy server: %v", err)

		return err
	}

	return nil
}
