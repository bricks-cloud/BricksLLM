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

type Provider interface {
	Incr(name string, tags []string, rate float64)
	Timing(name string, value time.Duration, tags []string, rate float64)
}

type Client struct {
	Provider Provider
}

var Singleton *Client

func Init(cfg *configPkg.Config) error {
	if cfg == nil {
		return errors.New("config is empty")
	}

	if cfg.TelemetryProvider == string(PROVIDER_DATADOG) {
		c, err := stats.InitializeClient(stats.Config{
			Enabled: cfg.StatsEnabled,
			Address: cfg.StatsAddress,
		})

		if err != nil {
			return err
		}

		Singleton = &Client{
			Provider: c,
		}

		return nil
	}

	if cfg.TelemetryProvider == string(PROVIDER_PROMETHEUS) {
		p, err := prometheus.Init(prometheus.Config{
			Enabled: cfg.PrometheusEnabled,
			Port:    cfg.PrometheusPort,
		})

		if err != nil {
			return err
		}

		Singleton = &Client{
			Provider: p,
		}

		return nil
	}

	return errors.New("unsupported telemetry provider")
}

func Incr(name string, tags []string, rate float64) {
	if Singleton != nil {
		Singleton.Provider.Incr(name, tags, rate)
	}
}

func Timing(name string, value time.Duration, tags []string, rate float64) {
	if Singleton != nil {
		Singleton.Provider.Timing(name, value, tags, rate)
	}
}
