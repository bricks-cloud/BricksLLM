package config

import (
	"fmt"
	"log"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	PostgresqlHosts               string
	PostgresqlDbName              string
	PostgresqlUsername            string
	PostgresqlPassword            string
	PostgresqlSslMode             string
	PostgresqlPort                string
	RedisHosts                    string
	RedisPort                     string
	RedisUsername                 string
	RedisPassword                 string
	RedisReadTimeout              time.Duration
	RedisWriteTimeout             time.Duration
	PostgresqlReadTimeout         time.Duration
	PostgresqlWriteTimeout        time.Duration
	InMemoryDbUpdateInterval      time.Duration
	OpenAiKey                     string
	StatsProvider                 string
	AdminPass                     string
	ProxyTimeout                  time.Duration
	NumberOfEventMessageConsumers int
	OpenAiApiKey                  string
	CustomPolicyDetectionTimeout  time.Duration
	AmazonRegion                  string
	AmazonRequestTimeout          time.Duration
	AmazonConnectionTimeout       time.Duration
}

var k = koanf.New(".")

func ParseEnvVariables() (*Config, error) {

	f := file.Provider("config.json")
	if err := k.Load(f, json.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}
	k.Load(env.Provider("", ".", func(s string) string {
		return s
	}), nil)

	f.Watch(func(event interface{}, err error) {
		if err != nil {
			log.Printf("watch error: %v", err)
			return
		}

		// Throw away the old config and load a fresh copy.
		log.Println("config changed. Reloading ...")
		k = koanf.New(".")
		k.Load(f, json.Parser())
		k.Print()
	})

	fmt.Println("name is = ", k.Duration("RedisWriteTimeout").String())

	cfg := &Config{}
	k.Unmarshal("", cfg)

	return cfg, nil
}
