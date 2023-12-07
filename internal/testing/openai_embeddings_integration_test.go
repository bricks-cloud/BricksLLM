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
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func embeddingsRequest(request *goopenai.EmbeddingRequest, apiKey string) (int, []byte, error) {
	jsonData, err := json.Marshal(request)

	if err != nil {
		return 0, nil, err
	}

	header := map[string][]string{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", apiKey)},
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8002", Path: "/api/providers/openai/v1/embeddings"},
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

func TestOpenAi_Embeddings(t *testing.T) {
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

		request := &goopenai.EmbeddingRequest{
			Model: goopenai.AdaEmbeddingV2,
			Input: "hello",
		}

		code, bs, err := embeddingsRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		request = &goopenai.EmbeddingRequest{
			Model: goopenai.AdaEmbeddingV2,
			Input: []string{
				"hello",
				"hello",
			},
		}

		code, bs, err = embeddingsRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		request = &goopenai.EmbeddingRequest{
			Model: goopenai.AdaEmbeddingV2,
			Input: []string{
				"hello",
				"hello",
			},
			EncodingFormat: goopenai.EmbeddingEncodingFormatBase64,
		}

		code, bs, err = embeddingsRequest(request, requestKey.Key)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))
	})
}
