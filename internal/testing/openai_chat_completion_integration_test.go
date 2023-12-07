package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/caarlos0/env"
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type config struct {
	OpenAiKey    string `env:"OPENAI_API_KEY" envDefault:"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"`
	AnthropicKey string `env:"ANTHROPIC_API_KEY" envDefault:"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"`
}

func parseEnvVariables() (*config, error) {
	cfg := &config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func chatCompletionRequest(request *goopenai.ChatCompletionRequest, apiKey string, customId string) (int, []byte, error) {
	jsonData, err := json.Marshal(request)

	if err != nil {
		return 0, nil, err
	}

	header := map[string][]string{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", apiKey)},
	}

	if len(customId) != 0 {
		header["X-CUSTOM-EVENT-ID"] = []string{customId}
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8002", Path: "/api/providers/openai/v1/chat/completions"},
		Header: header,
		Body:   io.NopCloser(bytes.NewBuffer(jsonData)),
	})

	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, bs, err
}

func completionRequest(request *anthropic.CompletionRequest, apiKey string, customId string) (int, []byte, error) {
	jsonData, err := json.Marshal(request)

	if err != nil {
		return 0, nil, err
	}

	header := map[string][]string{
		"content-type":      {"application/json"},
		"accept":            {"application/json"},
		"x-api-key":         {apiKey},
		"anthropic-version": {"2023-06-01"},
	}

	if len(customId) != 0 {
		header["X-CUSTOM-EVENT-ID"] = []string{customId}
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8002", Path: "/api/providers/anthropic/v1/complete"},
		Header: header,
		Body:   io.NopCloser(bytes.NewBuffer(jsonData)),
	})

	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, bs, err
}

func TestOpenAi_ChatCompletion(t *testing.T) {
	c, _ := parseEnvVariables()
	db := connectToPostgreSqlDb()

	t.Run("when api key is valid", func(t *testing.T) {
		defer deleteEventsTable(db)

		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": c.OpenAiKey,
			},
			Name: "test",
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)
		defer deleteProviderSetting(db, created.Id)

		requestKey := &key.RequestKey{
			Name:      "Spike's Testing Key",
			Tags:      []string{"spike"},
			Key:       "actualKey",
			SettingId: created.Id,
		}

		createdKey, err := createApiKey(requestKey)
		require.Nil(t, err)
		defer deleteApiKey(db, createdKey.KeyId)

		time.Sleep(6 * time.Second)

		request := &goopenai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []goopenai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}

		code, bs, err := chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))
	})
}
