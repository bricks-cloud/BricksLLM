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
	RedisReadTimeout         time.Duration `env:"REDIS_READ_TIME_OUT" envDefault:"2s"`
	RedisWriteTimeout        time.Duration `env:"REDIS_WRITE_TIME_OUT" envDefault:"1s"`
	PostgresqlReadTimeout    time.Duration `env:"POSTGRESQL_READ_TIME_OUT" envDefault:"5s"`
	PostgresqlWriteTimeout   time.Duration `env:"POSTGRESQL_WRITE_TIME_OUT" envDefault:"5s"`
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
