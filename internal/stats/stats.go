package stats

import (
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

type Client struct {
	statsdc *statsd.Client
}

var instance *Client

func InitializeClient(address string) error {
	if instance == nil {
		instance = &Client{}
		statsd, err := statsd.New(address)
		if err != nil {
			return err
		}
		instance.statsdc = statsd

		return nil
	}

	return nil
}

func Incr(name string, tags []string, rate float64) {
	instance.statsdc.Incr(name, tags, rate)
}

func Timing(name string, value time.Duration, tags []string, rate float64) {
	instance.statsdc.Timing(name, value, tags, rate)
}
