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

func InitializeClient(cfg Config) (*Client, error) {
	statsd, err := statsd.New(cfg.Address)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:  cfg,
		statsdc: statsd,
	}, nil
}

func (c *Client) Incr(name string, tags []string, rate float64) {
	if c != nil && c.config.Enabled {
		c.statsdc.Incr(name, tags, rate)
	}
}

func (c *Client) Timing(name string, value time.Duration, tags []string, rate float64) {
	if c != nil && c.config.Enabled {
		c.statsdc.Timing(name, value, tags, rate)
	}
}
