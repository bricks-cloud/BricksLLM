package testing

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setUpApiKey(rk *key.RequestKey) (*key.ResponseKey, error) {
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
	t.Run("when Api key gets created", func(t *testing.T) {
		t.Skip()

		requestKey := &key.RequestKey{
			Name: "Spike's Testing Key",
			Tags: []string{"spike"},
			Key:  "actualKey",
		}

		responseKey, err := setUpApiKey(requestKey)
		require.Nil(t, err)

		assert.Equal(t, requestKey.Name, responseKey.Name)
		assert.Equal(t, requestKey.Tags, responseKey.Tags)
		assert.NotEmpty(t, responseKey.CreatedAt)
		assert.NotEmpty(t, responseKey.UpdatedAt)
		assert.NotEmpty(t, responseKey.KeyId)
		assert.False(t, responseKey.Revoked)
	})
}
