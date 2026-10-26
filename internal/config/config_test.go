package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfig_RedisSslEnvDefaults(t *testing.T) {
	os.Clearenv()
	cfg, err := LoadConfig(zap.NewNop())
	require.NoError(t, err)
	assert.False(t, cfg.RedisSslEnabled)
	assert.False(t, cfg.RedisSslInsecureSkipVerify)
}

func TestConfig_RedisSslEnvParsed(t *testing.T) {
	os.Clearenv()
	t.Setenv("REDIS_SSL_ENABLED", "true")
	t.Setenv("REDIS_SSL_INSECURE_SKIP_VERIFY", "true")

	cfg, err := LoadConfig(zap.NewNop())
	require.NoError(t, err)
	assert.True(t, cfg.RedisSslEnabled)
	assert.True(t, cfg.RedisSslInsecureSkipVerify)
}
