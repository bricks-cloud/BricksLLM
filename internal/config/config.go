package config

import (
	"github.com/caarlos0/env"
)

type Config struct {
	PostgresqlHosts      string `env:"HOSTS" envSeparator:":" envDefault:"localhost"`
	PostgresqlUsername   string `env:"POSTGRESQL_USERNAME"`
	PostgresqlPassword   string `env:"POSTGRESQL_PASSWORD"`
	PostgresqlSslEnabled bool   `env:"POSTGRESQL_SSL_ENABLED" envDefault:"false"`
	PostgresqlPort       string `env:"POSTGRESQL_PORT" envDefault:"5432"`
	EncryptionKey        string `env:"ENCRYPTION_KEY,required"`
}

func ParseEnvVariables() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
