package testing

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

func deleteApiKey(db *sql.DB, id string) error {
	_, err := db.ExecContext(context.Background(), "DELETE FROM keys WHERE $1 = key_id", id)
	return err
}

func getApiKeys(tags []string, provider string) ([]*key.ResponseKey, error) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/key-management/keys"},
	}

	q := req.URL.Query()
	if len(tags) == 1 {
		q.Add("tag", tags[0])
	} else if len(tags) >= 2 {
		for _, tag := range tags {
			q.Add("tags", tag)
		}
	}

	if len(provider) != 0 {
		q.Add("provider", provider)
	}

	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var retrieved []*key.ResponseKey

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &retrieved); err != nil {
		return nil, err
	}

	return retrieved, nil
}

func createApiKey(rk *key.RequestKey) (*key.ResponseKey, error) {
	jsonData, err := json.Marshal(rk)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPut,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/key-management/keys"},
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respk key.ResponseKey

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &respk); err != nil {
		return nil, err
	}

	return &respk, nil
}

func TestApiKey_Creation(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when an api key gets created", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		key := &key.RequestKey{
			Name:      "Spike's Testing Key",
			Tags:      []string{"spike"},
			Key:       "actualKey",
			SettingId: created.Id,
		}

		createdKey, err := createApiKey(key)
		require.Nil(t, err)
		defer deleteApiKey(db, createdKey.KeyId)

		assert.Equal(t, key.Name, createdKey.Name)
		assert.Equal(t, key.Tags, createdKey.Tags)
		assert.NotEmpty(t, createdKey.CreatedAt)
		assert.NotEmpty(t, createdKey.UpdatedAt)
		assert.NotEmpty(t, createdKey.KeyId)
		assert.False(t, createdKey.Revoked)
	})
}

func TestApiKey_Retrieval(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when retrieving api keys by tags", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)
		defer deleteProviderSetting(db, created.Id)

		reqkey := &key.RequestKey{
			Name:      "Spike's Testing Key",
			Tags:      []string{"tag-1"},
			Key:       "spike's key",
			SettingId: created.Id,
		}

		createdKey, err := createApiKey(reqkey)
		require.Nil(t, err)
		defer deleteApiKey(db, createdKey.KeyId)

		reqkey = &key.RequestKey{
			Name:      "Donovan's Testing Key",
			Tags:      []string{"tag-1", "tag-2"},
			Key:       "donovan's key",
			SettingId: created.Id,
		}

		createdSecondKey, err := createApiKey(reqkey)
		require.Nil(t, err)
		defer deleteApiKey(db, createdSecondKey.KeyId)

		retrieved, err := getApiKeys([]string{
			"tag-1",
		}, "")
		require.Nil(t, err)
		assert.Equal(t, 2, len(retrieved))

		retrieved, err = getApiKeys([]string{
			"tag-2",
		}, "")
		require.Nil(t, err)
		assert.Equal(t, 1, len(retrieved))

		retrieved, err = getApiKeys([]string{
			"tag-3",
		}, "")
		require.Nil(t, err)
		assert.Equal(t, 0, len(retrieved))

		retrieved, err = getApiKeys([]string{
			"tag-1",
			"tag-2",
		}, "")
		require.Nil(t, err)
		assert.Equal(t, 1, len(retrieved))
	})

	t.Run("when retrieving api keys by provider", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)
		defer deleteProviderSetting(db, created.Id)

		reqkey := &key.RequestKey{
			Name:      "Spike's Testing Key",
			Tags:      []string{"tag-1"},
			Key:       "spike's key",
			SettingId: created.Id,
		}

		createdKey, err := createApiKey(reqkey)
		require.Nil(t, err)
		defer deleteApiKey(db, createdKey.KeyId)

		reqkey = &key.RequestKey{
			Name:      "Donovan's Testing Key",
			Tags:      []string{"tag-1", "tag-2"},
			Key:       "donovan's key",
			SettingId: created.Id,
		}

		createdSecondKey, err := createApiKey(reqkey)
		require.Nil(t, err)
		defer deleteApiKey(db, createdSecondKey.KeyId)

		retrieved, err := getApiKeys([]string{}, "openai")
		require.Nil(t, err)
		assert.Equal(t, 2, len(retrieved))

		retrieved, err = getApiKeys([]string{}, "anthropic")
		require.Nil(t, err)
		assert.Equal(t, 0, len(retrieved))
	})
}

func TestKey_AccessControl(t *testing.T) {
	c, _ := parseEnvVariables()
	db := connectToPostgreSqlDb()

	defer deleteEventsTable(db)

	t.Run("when request to proxy does not have an api key", func(t *testing.T) {
		request := &goopenai.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []goopenai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}
		code, _, err := chatCompletionRequest(request, "", "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("when api key is valid", func(t *testing.T) {
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

	t.Run("when api key has a ttl", func(t *testing.T) {
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
			Ttl:       "10s",
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

		time.Sleep(4 * time.Second)

		code, bs, err = chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, code, string(bs))
	})

	t.Run("when api key has a total spend limit", func(t *testing.T) {
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
			Name:           "Spike's Testing Key",
			Tags:           []string{"spike"},
			Key:            "actualKey",
			CostLimitInUsd: 0.00007,
			SettingId:      created.Id,
		}

		createdKey, err := createApiKey(requestKey)
		require.Nil(t, err)
		defer deleteApiKey(db, createdKey.KeyId)

		time.Sleep(6 * time.Second)
		request := &goopenai.ChatCompletionRequest{
			Model: "gpt-3.5-turbo",
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

		code, bs, err = chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		code, bs, err = chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusUnauthorized, code, string(bs))
	})
}
