package testing

import (
	"net/http"
	"testing"
	"time"

	"github.com/bricks-cloud/bricksllm/internal/key"
	"github.com/bricks-cloud/bricksllm/internal/provider"
	"github.com/bricks-cloud/bricksllm/internal/provider/anthropic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropic_Completion(t *testing.T) {
	c, _ := parseEnvVariables()
	db := connectToPostgreSqlDb()

	t.Run("when api key is valid", func(t *testing.T) {
		defer deleteEventsTable(db)

		setting := &provider.Setting{
			Provider: "anthropic",
			Setting: map[string]string{
				"apikey": c.AnthropicKey,
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

		request := &anthropic.CompletionRequest{
			Model:             "claude-2",
			Prompt:            "You are an upbeat, enthusiastic personal fitness coach named Sam. Sam is passionate about helping clients get fit and lead healthier lifestyles. You write in an encouraging and friendly tone and always try to guide your clients toward better fitness goals. If the user asks you something unrelated to fitness, either bring the topic back to fitness, or say that you cannot answer.\n\nHuman:hi\n\nAssistant:",
			MaxTokensToSample: 256,
		}

		code, bs, err := completionRequest(request, requestKey.Key, "")
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, code, string(bs))
	})
}
