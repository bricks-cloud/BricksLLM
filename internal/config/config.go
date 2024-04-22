package config

import (
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	PostgresqlHosts               string        `env:"POSTGRESQL_HOSTS" envSeparator:":" envDefault:"localhost"`
	PostgresqlDbName              string        `env:"POSTGRESQL_DB_NAME"`
	PostgresqlUsername            string        `env:"POSTGRESQL_USERNAME"`
	PostgresqlPassword            string        `env:"POSTGRESQL_PASSWORD"`
	PostgresqlSslMode             string        `env:"POSTGRESQL_SSL_MODE" envDefault:"disable"`
	PostgresqlPort                string        `env:"POSTGRESQL_PORT" envDefault:"5432"`
	RedisHosts                    string        `env:"REDIS_HOSTS" envSeparator:":" envDefault:"localhost"`
	RedisPort                     string        `env:"REDIS_PORT" envDefault:"6379"`
	RedisUsername                 string        `env:"REDIS_USERNAME"`
	RedisPassword                 string        `env:"REDIS_PASSWORD"`
	RedisReadTimeout              time.Duration `env:"REDIS_READ_TIME_OUT" envDefault:"1s"`
	RedisWriteTimeout             time.Duration `env:"REDIS_WRITE_TIME_OUT" envDefault:"500ms"`
	PostgresqlReadTimeout         time.Duration `env:"POSTGRESQL_READ_TIME_OUT" envDefault:"2m"`
	PostgresqlWriteTimeout        time.Duration `env:"POSTGRESQL_WRITE_TIME_OUT" envDefault:"5s"`
	InMemoryDbUpdateInterval      time.Duration `env:"IN_MEMORY_DB_UPDATE_INTERVAL" envDefault:"5s"`
	OpenAiKey                     string        `env:"OPENAI_API_KEY"`
	StatsProvider                 string        `env:"STATS_PROVIDER"`
	AdminPass                     string        `env:"ADMIN_PASS"`
	ProxyTimeout                  time.Duration `env:"PROXY_TIMEOUT" envDefault:"600s"`
	NumberOfEventMessageConsumers int           `env:"NUMBER_OF_EVENT_MESSAGE_CONSUMERS" envDefault:"3"`
	OpenAiApiKey                  string        `env:"OPENAI_API_KEY"`
	CustomPolicyDetectionTimeout  time.Duration `env:"CUSTOM_POLICY_DETECTION_TIMEOUT" envDefault:"10m"`
	AmazonRegion                  string        `env:"AMAZON_REGION" envDefault:"us-west-2"`
	AmazonRequestTimeout          time.Duration `env:"AMAZON_REQUEST_TIMEOUT" envDefault:"5s"`
	AmazonConnectionTimeout       time.Duration `env:"AMAZON_CONNECTION_TIMEOUT" envDefault:"10s"`
}

func ParseEnvVariables() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
