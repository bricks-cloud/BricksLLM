package telemetry

import (
	"errors"
	"time"

	configPkg "github.com/bricks-cloud/bricksllm/internal/config"
	"github.com/bricks-cloud/bricksllm/internal/telemetry/prometheus"
	"github.com/bricks-cloud/bricksllm/internal/telemetry/stats"
)

type ProviderType string

const (
	PROVIDER_DATADOG    ProviderType = "statsd"
	PROVIDER_PROMETHEUS ProviderType = "prometheus"
)

type Config struct {
	Provider      ProviderType
	statsdCfg     stats.Config
	prometheusCfg prometheus.Config
}

var (
	config Config
)

func Init(cfg *configPkg.Config) error {
	config = Config{
		Provider: ProviderType(cfg.TelemetryProvider),
		statsdCfg: stats.Config{
			Enabled: cfg.StatsEnabled,
			Address: cfg.StatsAddress,
		},
		prometheusCfg: prometheus.Config{
			Enabled: cfg.PrometheusEnabled,
			Port:    cfg.PrometheusPort,
		},
	}

	if config.Provider == PROVIDER_DATADOG {
		stats.InitializeClient(config.statsdCfg)
		return nil
	}

	if config.Provider == PROVIDER_PROMETHEUS {
		prometheus.Init(config.prometheusCfg)
		return nil
	}
	return errors.New("unsupported telemetry provider")
}

func Incr(name string, tags []string, rate float64) {
	if config.Provider == PROVIDER_DATADOG {
		stats.Incr(name, tags, rate)
		return
	}
	if config.Provider == PROVIDER_PROMETHEUS {
		prometheus.Incr(name, tags, rate)
		return
	}
}

func Timing(name string, value time.Duration, tags []string, rate float64) {
	if config.Provider == PROVIDER_DATADOG {
		stats.Timing(name, value, tags, rate)
		return
	}
	if config.Provider == PROVIDER_PROMETHEUS {
		prometheus.Timing(name, value, tags, rate)
		return
	}
}
