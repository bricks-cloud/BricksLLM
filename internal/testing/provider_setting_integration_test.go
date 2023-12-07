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

func deleteProviderSetting(db *sql.DB, id string) error {
	_, err := db.ExecContext(context.Background(), "DELETE FROM provider_settings WHERE $1 = id", id)

	return err
}

func createProviderSetting(s *provider.Setting) (*provider.Setting, error) {
	jsonData, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPut,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/provider-settings"},
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var created provider.Setting

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &created); err != nil {
		return nil, err
	}

	return &created, nil
}

func getProviderSettings() ([]*provider.Setting, error) {
	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/provider-settings"},
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var settings []*provider.Setting

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func updateProviderSetting(id string, setting *provider.UpdateSetting) (*provider.Setting, error) {
	jsonData, err := json.Marshal(setting)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPatch,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/provider-settings/" + id},
		Body:   io.NopCloser(bytes.NewBuffer(jsonData)),
	})

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updated *provider.Setting

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &updated); err != nil {
		return nil, err
	}

	return updated, nil
}

func TestProviderSetting_Creation(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when a provider setting gets created", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
			AllowedModels: []string{
				"gpt-3.5-turbo",
			},
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		defer deleteProviderSetting(db, created.Id)

		assert.Equal(t, setting.Name, created.Name)
		assert.Equal(t, setting.Provider, created.Provider)
		assert.Nil(t, created.Setting)
		assert.NotEmpty(t, created.CreatedAt)
		assert.NotEmpty(t, created.UpdatedAt)
		assert.NotEmpty(t, created.Id)
		assert.NotEmpty(t, created.AllowedModels)
	})
}

func TestProviderSetting_AllowedModels(t *testing.T) {
	c, _ := parseEnvVariables()
	db := connectToPostgreSqlDb()

	t.Run("when allowed models are set", func(t *testing.T) {
		defer deleteEventsTable(db)

		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": c.OpenAiKey,
			},
			Name: "test",
			AllowedModels: []string{
				"gpt-3.5-turbo",
			},
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
		assert.Equal(t, http.StatusForbidden, code, string(bs))

		request = &goopenai.ChatCompletionRequest{
			Model: "gpt-3.5-turbo",
			Messages: []goopenai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "hi",
				},
			},
		}

		code, bs, err = chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))
	})
}

func TestProviderSetting_Retrieval(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when retrieving created provider settings", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
			AllowedModels: []string{
				"gpt-3.5-turbo",
			},
		}

		settingOne := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test-1",
			AllowedModels: []string{
				"gpt-3.5-turbo",
			},
		}

		settingMap := map[string]*provider.Setting{}
		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		createdOne, err := createProviderSetting(settingOne)
		require.Nil(t, err)

		settings, err := getProviderSettings()
		require.Nil(t, err)

		for _, setting := range settings {
			settingMap[setting.Id] = setting
		}

		_, createdExists := settingMap[created.Id]
		assert.True(t, createdExists)

		_, createdOneExists := settingMap[createdOne.Id]
		assert.True(t, createdOneExists)

		for _, setting := range settings {
			deleteProviderSetting(db, setting.Id)
		}

	})
}

func TestProviderSetting_Update(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when updating provider settings name", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		name := ""
		updates := &provider.UpdateSetting{
			Name: &name,
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		defer deleteProviderSetting(db, created.Id)

		updated, err := updateProviderSetting(created.Id, updates)
		require.Nil(t, err)

		assert.Equal(t, *updates.Name, updated.Name)
		assert.NotEqual(t, created.UpdatedAt, updates.UpdatedAt)
	})

	t.Run("when updating provider settings with incorrect settings", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		name := "test-1"
		updates := &provider.UpdateSetting{
			Setting: map[string]string{
				"api": "secret-key",
			},
			Name: &name,
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		defer deleteProviderSetting(db, created.Id)

		_, err = updateProviderSetting(created.Id, updates)
		assert.Error(t, err)
	})

	t.Run("when updating provider settings name and setting", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		name := "test-1"
		updates := &provider.UpdateSetting{
			Setting: map[string]string{
				"apikey": "secret-key-1",
			},
			Name: &name,
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		defer deleteProviderSetting(db, created.Id)

		updated, err := updateProviderSetting(created.Id, updates)
		require.NoError(t, err)

		assert.Equal(t, *updates.Name, updated.Name)
		assert.NotEqual(t, created.UpdatedAt, updates.UpdatedAt)
	})

	t.Run("when updating provider settings allowed models", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		updates := &provider.UpdateSetting{
			AllowedModels: &[]string{
				"/api/providers/openai/",
			},
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		defer deleteProviderSetting(db, created.Id)

		updated, err := updateProviderSetting(created.Id, updates)
		require.Nil(t, err)

		assert.ElementsMatch(t, *updates.AllowedModels, updated.AllowedModels)
		assert.NotEqual(t, created.UpdatedAt, updates.UpdatedAt)
	})

	t.Run("when updating provider settings allowed models to empty", func(t *testing.T) {
		setting := &provider.Setting{
			Provider: "openai",
			Setting: map[string]string{
				"apikey": "secret-key",
			},
			Name: "test",
		}

		updates := &provider.UpdateSetting{
			AllowedModels: &[]string{},
		}

		created, err := createProviderSetting(setting)
		require.Nil(t, err)

		defer deleteProviderSetting(db, created.Id)

		updated, err := updateProviderSetting(created.Id, updates)
		require.Nil(t, err)

		assert.Empty(t, updated.AllowedModels)
		assert.NotEqual(t, created.UpdatedAt, updates.UpdatedAt)
	})
}
