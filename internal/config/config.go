package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	PostgresqlHosts               string        `koanf:"postgresql_hosts"`
	PostgresqlDbName              string        `koanf:"postgresql_db_name"`
	PostgresqlUsername            string        `koanf:"postgresql_username"`
	PostgresqlPassword            string        `koanf:"postgresql_password"`
	PostgresqlSslMode             string        `koanf:"postgresql_ssl_mode"`
	PostgresqlPort                string        `koanf:"postgresql_port"`
	RedisHosts                    string        `koanf:"redis_hosts"`
	RedisPort                     string        `koanf:"redis_port"`
	RedisUsername                 string        `koanf:"redis_username"`
	RedisPassword                 string        `koanf:"redis_password"`
	RedisReadTimeout              time.Duration `koanf:"redis_read_time_out"`
	RedisWriteTimeout             time.Duration `koanf:"redis_write_time_out"`
	PostgresqlReadTimeout         time.Duration `koanf:"postgresql_read_time_out"`
	PostgresqlWriteTimeout        time.Duration `koanf:"postgresql_write_time_out"`
	InMemoryDbUpdateInterval      time.Duration `koanf:"in_memory_db_update_interval"`
	OpenAiKey                     string        `koanf:"openai_key"`
	StatsProvider                 string        `koanf:"stats_provider"`
	AdminPass                     string        `koanf:"admin_pass"`
	ProxyTimeout                  time.Duration `koanf:"proxy_timeout"`
	NumberOfEventMessageConsumers int           `koanf:"number_of_event_message_consumers"`
	OpenAiApiKey                  string        `koanf:"openai_api_key"`
	CustomPolicyDetectionTimeout  time.Duration `koanf:"custom_policy_detection_timeout"`
	AmazonRegion                  string        `koanf:"amazon_region"`
	AmazonRequestTimeout          time.Duration `koanf:"amazon_request_timeout"`
	AmazonConnectionTimeout       time.Duration `koanf:"amazon_connection_timeout"`
}

func prepareDotEnv(envFilePath string) error {
	if envFilePath == "" {
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath := filepath.Dir(ex)

		// first check .env, then .env_{DEV|TEST|UAT|PROD}
		envFile := exPath + "/.env"
		envFilePath = envFile
	}

	err := godotenv.Load(envFilePath)
	return err
}

var k = koanf.New(".")

func LoadConfig() (*Config, error) {
	err := prepareDotEnv(".env")
	if err != nil {
		panic(err)
	}

	cfgPath := os.Getenv("CONFIG_FILE_NAME")
	if cfgPath != "" {
		f := file.Provider(cfgPath)
		if err := k.Load(f, json.Parser()); err != nil {
			log.Fatalf("error loading config from json file: %v", err)
		}
	}

	k.Load(env.Provider("", ".", func(s string) string {
		return strings.ToLower(s)
	}), nil)

	cfg := &Config{}
	k.Unmarshal("", cfg)

	return cfg, nil
}
