package main

import (
	"testing"

	"github.com/bricks-cloud/bricksllm/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRedisOptions_DefaultDisabled(t *testing.T) {
	cfg := &config.Config{
		RedisHosts:        "redis.internal",
		RedisPort:         "6380",
		RedisUsername:     "default",
		RedisPassword:     "secretpass",
		RedisDBStartIndex: 2,
		RedisSslEnabled:   false,
	}

	opts := buildRedisOptions(cfg, 3)
	require.NotNil(t, opts)
	assert.Equal(t, "redis.internal:6380", opts.Addr)
	assert.Equal(t, "default", opts.Username)
	assert.Equal(t, "secretpass", opts.Password)
	assert.Equal(t, 5, opts.DB)
	assert.Nil(t, opts.TLSConfig)
}

func TestBuildRedisOptions_SslEnabled(t *testing.T) {
	cfg := &config.Config{
		RedisHosts:                 "secure-redis.cloud",
		RedisPort:                  "6379",
		RedisUsername:              "app",
		RedisPassword:              "tls-secret",
		RedisDBStartIndex:          0,
		RedisSslEnabled:            true,
		RedisSslInsecureSkipVerify: false,
	}

	opts := buildRedisOptions(cfg, 1)
	require.NotNil(t, opts)
	assert.Equal(t, "secure-redis.cloud:6379", opts.Addr)
	assert.Equal(t, "app", opts.Username)
	assert.Equal(t, "tls-secret", opts.Password)
	assert.Equal(t, 1, opts.DB)
	require.NotNil(t, opts.TLSConfig)
	assert.False(t, opts.TLSConfig.InsecureSkipVerify)
}

func TestBuildRedisOptions_SslWithInsecureSkipVerify(t *testing.T) {
	cfg := &config.Config{
		RedisHosts:                 "self-signed.local",
		RedisPort:                  "6379",
		RedisSslEnabled:            true,
		RedisSslInsecureSkipVerify: true,
	}

	opts := buildRedisOptions(cfg, 0)
	require.NotNil(t, opts)
	require.NotNil(t, opts.TLSConfig)
	assert.True(t, opts.TLSConfig.InsecureSkipVerify)
}
