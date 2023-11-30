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

	"github.com/bricks-cloud/bricksllm/internal/provider/custom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deleteCustomProvider(db *sql.DB, id string) error {
	_, err := db.ExecContext(context.Background(), "DELETE FROM custom_providers WHERE $1 = key_id", id)
	return err
}

func getCustomProviders(tags []string, provider string) ([]*custom.Provider, error) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/custom/providers"},
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var retrieved []*custom.Provider

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

func updateCustomProvider(id string, provider *custom.Provider) (*custom.Provider, error) {
	jsonData, err := json.Marshal(provider)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method: http.MethodPatch,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/custom/providers/" + id},
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updated *custom.Provider

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

func createCustomProvider(provider *custom.Provider) (*custom.Provider, error) {
	jsonData, err := json.Marshal(provider)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/custom/providers"},
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: io.NopCloser(bytes.NewBuffer(jsonData)),
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updated *custom.Provider

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

func TestCustomProvider_Creation(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when a custom provider gets created", func(t *testing.T) {
		rc := &custom.RouteConfig{
			Path:                             "/chat/completion",
			TargetUrl:                        "https://api.openai.com/v1/chat/completions",
			StreamLocation:                   "stream",
			StreamResponseCompletionLocation: "choices.#.delta.content",
			StreamEndWord:                    "choices.#.delta.content",
			StreamMaxEmptyMessages:           10,
			ModelLocation:                    "model",
			RequestPromptLocation:            "messages.#.content",
			ResponseCompletionLocation:       "choices.#.message.content",
		}
		cp := &custom.Provider{
			Provider: "chima",
			RouteConfigs: []*custom.RouteConfig{
				rc,
			},
		}

		created, err := createCustomProvider(cp)
		require.Nil(t, err)

		require.NotEmpty(t, created.Id)
		require.NotEmpty(t, created.CreatedAt)
		require.NotEmpty(t, created.UpdatedAt)

		defer deleteCustomProvider(db, created.Id)
		assert.Equal(t, cp.Provider, created.Provider)

		require.Equal(t, len(cp.RouteConfigs), len(created.RouteConfigs))

		createdRc := created.RouteConfigs[0]
		assert.Equal(t, rc.Path, createdRc.Path)
		assert.Equal(t, rc.TargetUrl, createdRc.TargetUrl)
		assert.Equal(t, rc.StreamLocation, createdRc.StreamLocation)
		assert.Equal(t, rc.StreamResponseCompletionLocation, createdRc.StreamResponseCompletionLocation)
		assert.Equal(t, rc.StreamEndWord, createdRc.StreamEndWord)
		assert.Equal(t, rc.StreamMaxEmptyMessages, createdRc.StreamMaxEmptyMessages)
		assert.Equal(t, rc.ModelLocation, createdRc.ModelLocation)
		assert.Equal(t, rc.RequestPromptLocation, createdRc.RequestPromptLocation)
		assert.Equal(t, rc.ResponseCompletionLocation, createdRc.ResponseCompletionLocation)
	})
}

func TestCustomProvider_Update(t *testing.T) {
	db := connectToPostgreSqlDb()
	t.Run("when a custom provider gets created", func(t *testing.T) {
		rc := &custom.RouteConfig{
			Path:                             "/chat/completion",
			TargetUrl:                        "https://api.openai.com/v1/chat/completions",
			StreamLocation:                   "stream",
			StreamResponseCompletionLocation: "choices.#.delta.content",
			StreamEndWord:                    "choices.#.delta.content",
			StreamMaxEmptyMessages:           10,
			ModelLocation:                    "model",
			RequestPromptLocation:            "messages.#.content",
			ResponseCompletionLocation:       "choices.#.message.content",
		}
		cp := &custom.Provider{
			Provider: "chima",
			RouteConfigs: []*custom.RouteConfig{
				rc,
			},
		}

		created, err := createCustomProvider(cp)
		require.Nil(t, err)

		require.NotEmpty(t, created.Id)
		require.NotEmpty(t, created.CreatedAt)
		require.NotEmpty(t, created.UpdatedAt)

		defer deleteCustomProvider(db, created.Id)
		assert.Equal(t, cp.Provider, created.Provider)

		require.Equal(t, len(cp.RouteConfigs), len(created.RouteConfigs))

		createdRc := created.RouteConfigs[0]
		assert.Equal(t, rc.Path, createdRc.Path)
		assert.Equal(t, rc.TargetUrl, createdRc.TargetUrl)
		assert.Equal(t, rc.StreamLocation, createdRc.StreamLocation)
		assert.Equal(t, rc.StreamResponseCompletionLocation, createdRc.StreamResponseCompletionLocation)
		assert.Equal(t, rc.StreamEndWord, createdRc.StreamEndWord)
		assert.Equal(t, rc.StreamMaxEmptyMessages, createdRc.StreamMaxEmptyMessages)
		assert.Equal(t, rc.ModelLocation, createdRc.ModelLocation)
		assert.Equal(t, rc.RequestPromptLocation, createdRc.RequestPromptLocation)
		assert.Equal(t, rc.ResponseCompletionLocation, createdRc.ResponseCompletionLocation)
	})
}
