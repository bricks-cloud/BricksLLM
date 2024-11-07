package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/caarlos0/env"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
)

type Config struct {
	PostgresqlHosts               string        `koanf:"postgresql_hosts" env:"POSTGRESQL_HOSTS" envSeparator:":" envDefault:"localhost"`
	PostgresqlDbName              string        `koanf:"postgresql_db_name" env:"POSTGRESQL_DB_NAME"`
	PostgresqlUsername            string        `koanf:"postgresql_username" env:"POSTGRESQL_USERNAME"`
	PostgresqlPassword            string        `koanf:"postgresql_password" env:"POSTGRESQL_PASSWORD"`
	PostgresqlSslMode             string        `koanf:"postgresql_ssl_mode" env:"POSTGRESQL_SSL_MODE" envDefault:"disable"`
	PostgresqlPort                string        `koanf:"postgresql_port" env:"POSTGRESQL_PORT" envDefault:"5432"`
	RedisHosts                    string        `koanf:"redis_hosts" env:"REDIS_HOSTS" envSeparator:":" envDefault:"localhost"`
	RedisPort                     string        `koanf:"redis_port" env:"REDIS_PORT" envDefault:"6379"`
	RedisUsername                 string        `koanf:"redis_username" env:"REDIS_USERNAME"`
	RedisPassword                 string        `koanf:"redis_password" env:"REDIS_PASSWORD"`
	RedisInsecureSkipVerify       bool          `koanf:"redis_insecure_skip_verify" env:"REDIS_INSECURE_SKIP_VERIFY" envDefault:"false"`
	RedisDBStartIndex             int           `koanf:"redis_db_start_index" env:"REDIS_DB_START_INDEX" envDefault:"0"`
	RedisReadTimeout              time.Duration `koanf:"redis_read_time_out" env:"REDIS_READ_TIME_OUT" envDefault:"1s"`
	RedisWriteTimeout             time.Duration `koanf:"redis_write_time_out" env:"REDIS_WRITE_TIME_OUT" envDefault:"500ms"`
	PostgresqlReadTimeout         time.Duration `koanf:"postgresql_read_time_out" env:"POSTGRESQL_READ_TIME_OUT" envDefault:"10m"`
	PostgresqlWriteTimeout        time.Duration `koanf:"postgresql_write_time_out" env:"POSTGRESQL_WRITE_TIME_OUT" envDefault:"5s"`
	InMemoryDbUpdateInterval      time.Duration `koanf:"in_memory_db_update_interval" env:"IN_MEMORY_DB_UPDATE_INTERVAL" envDefault:"5s"`
	TelemetryProvider             string        `koanf:"telemetry_provider" env:"TELEMETRY_PROVIDER" envDefault:"statsd"`
	StatsEnabled                  bool          `koanf:"stats_enabled" env:"STATS_ENABLED" envDefault:"true"`
	StatsAddress                  string        `koanf:"stats_address" env:"STATS_ADDRESS" envDefault:"127.0.0.1:8125"`
	PrometheusEnabled             bool          `koanf:"prometheus_enabled" env:"PROMETHEUS_ENABLED" envDefault:"true"`
	PrometheusPort                string        `koanf:"prometheus_port" env:"PROMETHEUS_PORT" envDefault:"2112"`
	OpenTelemetryEnabled          bool          `koanf:"open_telemetry_enabled" env:"OPEN_TELEMETRY_ENABLED" envDefault:"false"`
	OpenTelemetryEndpoint         string        `koanf:"open_telemetry_endpoint" env:"OPEN_TELEMETRY_ENDPOINT" envDefault:"localhost:4318"`
	AdminPass                     string        `koanf:"admin_pass" env:"ADMIN_PASS"`
	ProxyTimeout                  time.Duration `koanf:"proxy_timeout" env:"PROXY_TIMEOUT" envDefault:"600s"`
	NumberOfEventMessageConsumers int           `koanf:"number_of_event_message_consumers" env:"NUMBER_OF_EVENT_MESSAGE_CONSUMERS" envDefault:"3"`
	OpenAiApiKey                  string        `koanf:"openai_api_key" env:"OPENAI_API_KEY"`
	CustomPolicyDetectionTimeout  time.Duration `koanf:"custom_policy_detection_timeout" env:"CUSTOM_POLICY_DETECTION_TIMEOUT" envDefault:"10m"`
	AmazonRegion                  string        `koanf:"amazon_region" env:"AMAZON_REGION" envDefault:"us-west-2"`
	AmazonRequestTimeout          time.Duration `koanf:"amazon_request_timeout" env:"AMAZON_REQUEST_TIMEOUT" envDefault:"5s"`
	AmazonConnectionTimeout       time.Duration `koanf:"amazon_connection_timeout" env:"AMAZON_CONNECTION_TIMEOUT" envDefault:"10s"`
	RemoveUserAgent               bool          `koanf:"remove_user_agent" env:"REMOVE_USER_AGENT" envDefault:"false"`
}

func prepareDotEnv(envFilePath string) error {
	err := godotenv.Load(envFilePath)
	if err != nil {
		ex, err := os.Executable()
		if err != nil {
			return err
		}

		exPath := filepath.Dir(ex)

		// first check .env, then .env_{DEV|TEST|UAT|PROD}
		envFile := exPath + "/.env"
		envFilePath = envFile

		err = godotenv.Load(envFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

var k = koanf.New(".")

func LoadConfig(log *zap.Logger) (*Config, error) {
	cfg := &Config{}

	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	err = prepareDotEnv(".env")
	if err != nil {
		log.Sugar().Infof("error loading config from .env file: %v", err)
	}

	cfgPath := os.Getenv("CONFIG_FILE_NAME")
	if cfgPath != "" {
		f := file.Provider(cfgPath)
		if err := k.Load(f, json.Parser()); err != nil {
			log.Sugar().Infof("error loading config from json file: %v", err)
		}
	}

	if len(cfgPath) != 0 {
		k.Unmarshal("", cfg)
	}

	return cfg, nil
}
