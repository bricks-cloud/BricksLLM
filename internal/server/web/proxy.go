package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

func NewProxyServer(log logger.Logger, m KeyManager, ks keyStorage, kms keyMemStorage, e estimator, v validator, r recorder, credential string, enc encrypter) (*ProxyServer, error) {
	router := gin.New()
	router.Use(getKeyValidator(kms, e, v, ks, log, enc))

	client := http.Client{}

	router.POST("/api/providers/openai", getOpenAiProxyHandler(r, credential, client, kms, log, enc))

	srv := &http.Server{
		Addr:    ":8002",
		Handler: router,
	}

	return &ProxyServer{
		logger: log,
		server: srv,
	}, nil
}

func getOpenAiProxyHandler(r recorder, credential string, client http.Client, kms keyMemStorage, log logger.Logger, enc encrypter) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", c.Request.Body)
		if err != nil {
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

		reqStart := time.Now()
		res, err := client.Do(req)
		if err != nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to send http request to openai")
			return
		}
		defer res.Body.Close()
		log.Debugf("record openai request latency %dms", time.Now().Sub(reqStart).Milliseconds())

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to read openai response body")
			return
		}

		if res.StatusCode == http.StatusOK {
			chatRes := &openai.ChatCompletionResponse{}
			err = json.Unmarshal(bytes, chatRes)
			if err != nil {
				JSON(c, http.StatusInternalServerError, "[BricksLLM] failed to parse openai response")
				return
			}

			keySpendStart := time.Now()
			err = r.RecordKeySpend(kc.KeyId, chatRes.Model, chatRes.Usage.PromptTokens, chatRes.Usage.CompletionTokens)
			if err != nil {
				log.Debugf("failed to record key spend for key: %s :%v", kc.KeyId, err)
			}

			log.Debugf("record key spend latency %dms", time.Now().Sub(keySpendStart).Milliseconds())
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
		ps.logger.Info("proxy server listening at 8002")
		ps.logger.Info("POST   :8002/api/providers/openai is ready for forwarding requests to openai")

		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ps.logger.Fatalf("error proxy server listening: %v", err)
			return
		}
	}()

}

func (ps *ProxyServer) Shutdown(ctx context.Context) error {
	if err := ps.server.Shutdown(ctx); err != nil {
		ps.logger.Debugf("error shutting down proxy server: %v", err)

		return err
	}

	return nil
}
