package testing

import (
	"bytes"
	"encoding/json"
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
		jsonData, err := json.Marshal(request)
		if err != nil {
			assert.Error(t, err)
		}

		resp, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Scheme: "http", Host: "localhost:8002", Path: "/api/providers/openai/v1/chat/completions"},
			Header: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: io.NopCloser(bytes.NewBuffer(jsonData)),
		})

		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
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

		time.Sleep(11 * time.Second)

		request := &openai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []openai.RequestMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}
		jsonData, err := json.Marshal(request)
		require.Nil(t, err)

		resp, err := http.DefaultClient.Do(&http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Scheme: "http", Host: "localhost:8002", Path: "/api/providers/openai/v1/chat/completions"},
			Header: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer actualKey"},
			},
			Body: io.NopCloser(bytes.NewBuffer(jsonData)),
		})

		require.Nil(t, err)

		defer resp.Body.Close()

		bs, err := io.ReadAll(resp.Body)
		require.Nil(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode, string(bs))
	})
}
