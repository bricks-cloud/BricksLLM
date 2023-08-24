package config

import (
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	PostgresqlHosts          string        `env:"POSTGRESQL_HOSTS" envSeparator:":" envDefault:"localhost"`
	PostgresqlUsername       string        `env:"POSTGRESQL_USERNAME"`
	PostgresqlPassword       string        `env:"POSTGRESQL_PASSWORD"`
	PostgresqlSslEnabled     bool          `env:"POSTGRESQL_SSL_ENABLED" envDefault:"false"`
	PostgresqlPort           string        `env:"POSTGRESQL_PORT" envDefault:"5432"`
	RedisHosts               string        `env:"REDIS_HOSTS" envSeparator:":" envDefault:"localhost"`
	RedisPort                string        `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword            string        `env:"REDIS_PASSWORD"`
	RedisReadTimeout         time.Duration `env:"REDIS_READ_TIME_OUT" envDefault:"1s"`
	RedisWriteTimeout        time.Duration `env:"REDIS_WRITE_TIME_OUT" envDefault:"500ms"`
	PostgresqlReadTimeout    time.Duration `env:"POSTGRESQL_READ_TIME_OUT" envDefault:"2s"`
	PostgresqlWriteTimeout   time.Duration `env:"POSTGRESQL_WRITE_TIME_OUT" envDefault:"1s"`
	InMemoryDbUpdateInterval time.Duration `env:"IN_MEMORY_DB_UPDATE_INTERVAL" envDefault:"10s"`
	OpenAiKey                string        `env:"OPENAI_API_KEY,required"`
}

func ParseEnvVariables() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
