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
	"github.com/bricks-cloud/bricksllm/internal/provider/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func proxyRequest(request *openai.ChatCompletionRequest, apiKey string) (int, []byte, error) {
	jsonData, err := json.Marshal(request)

	if err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8002", Path: "/api/providers/openai/v1/chat/completions"},
		Header: map[string][]string{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", apiKey)},
		},
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
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

func TestProxy_AccessControl(t *testing.T) {
	t.Run("when request to proxy does not have an api key", func(t *testing.T) {
		t.Skip()
		request := &openai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []openai.RequestMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}
		code, _, err := proxyRequest(request, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("when api key is valid", func(t *testing.T) {
		t.Skip()
		requestKey := &key.RequestKey{
			Name: "Spike's Testing Key",
			Tags: []string{"spike"},
			Key:  "actualKey",
		}

		_, err := setUpApiKey(requestKey)
		require.Nil(t, err)

		time.Sleep(2 * time.Second)

		request := &openai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []openai.RequestMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}

		code, bs, err := proxyRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))
	})

	t.Run("when api key has a ttl", func(t *testing.T) {
		t.Skip()
		requestKey := &key.RequestKey{
			Name: "Spike's Testing Key",
			Tags: []string{"spike"},
			Key:  "actualKey",
			Ttl:  "5s",
		}

		_, err := setUpApiKey(requestKey)
		require.Nil(t, err)

		time.Sleep(2 * time.Second)
		request := &openai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []openai.RequestMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}
		code, bs, err := proxyRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		time.Sleep(3 * time.Second)

		code, bs, err = proxyRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, code, string(bs))
	})

	t.Run("when api key has a total spend limit", func(t *testing.T) {
		requestKey := &key.RequestKey{
			Name:           "Spike's Testing Key",
			Tags:           []string{"spike"},
			Key:            "actualKey",
			CostLimitInUsd: 0.00007,
		}

		_, err := setUpApiKey(requestKey)
		require.Nil(t, err)

		time.Sleep(2 * time.Second)
		request := &openai.ChatCompletionRequest{
			Model: "gpt-3.5-turbo",
			Messages: []openai.RequestMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}
		code, bs, err := proxyRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		code, bs, err = proxyRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		code, bs, err = proxyRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, code, string(bs))
	})
}
