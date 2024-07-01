package stats

import (
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

type Config struct {
	Enabled bool
	Address string
}

type Client struct {
	config  Config
	statsdc *statsd.Client
}

var instance *Client

func InitializeClient(cfg Config) error {
	if instance == nil {
		instance = &Client{}
		instance.config = cfg

		statsd, err := statsd.New(cfg.Address)
		if err != nil {
			return err
		}
		instance.statsdc = statsd

		return nil
	}

	return nil
}

func Incr(name string, tags []string, rate float64) {
	if instance.config.Enabled {
		instance.statsdc.Incr(name, tags, rate)
	}
}

func Timing(name string, value time.Duration, tags []string, rate float64) {
	if instance.config.Enabled {
		instance.statsdc.Timing(name, value, tags, rate)
	}
}
