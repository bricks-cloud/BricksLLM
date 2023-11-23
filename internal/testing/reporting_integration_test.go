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

	"github.com/bricks-cloud/bricksllm/internal/event"
	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deleteEventsTable(db *sql.DB) error {
	_, err := db.ExecContext(context.Background(), "DELETE FROM events")
	return err
}

func getMetrics(rr *event.ReportingRequest) (*event.ReportingResponse, error) {
	jsonData, err := json.Marshal(rr)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(&http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/reporting/events"},
		Body:   io.NopCloser(bytes.NewBuffer(jsonData)),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metrics event.ReportingResponse
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

func getEvents(customId string) ([]*event.Event, error) {
	req := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: "localhost:8001", Path: "/api/events"},
	}

	q := req.URL.Query()
	if len(customId) != 0 {
		q.Add("customId", customId)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []*event.Event
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(data))
	}

	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}

	return events, nil
}

func TestReporting_Metrics(t *testing.T) {
	c, _ := parseEnvVariables()
	db := connectToPostgreSqlDb()
	defer deleteEventsTable(db)

	t.Run("when retrieving metrics", func(t *testing.T) {
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

		start := time.Now()

		code, bs, err := chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		code, bs, err = chatCompletionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		metrics, err := getMetrics(&event.ReportingRequest{
			Start:     start.Unix(),
			End:       start.Unix() + 10,
			Increment: 20,
		})
		require.Nil(t, err)
		require.Equal(t, 1, len(metrics.DataPoints))

		dp := metrics.DataPoints[0]

		require.Equal(t, int64(2), dp.NumberOfRequests)
	})
}

func TestReporting_EventRetrieval(t *testing.T) {
	c, _ := parseEnvVariables()
	db := connectToPostgreSqlDb()
	defer deleteEventsTable(db)

	t.Run("when retrieving an event", func(t *testing.T) {
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

		customId := "custom-id"
		code, bs, err := chatCompletionRequest(request, requestKey.Key, customId)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))

		events, err := getEvents(customId)
		require.Nil(t, err)

		require.Equal(t, 1, len(events))
		event := events[0]
		assert.Equal(t, http.StatusOK, event.Status)
	})
}
