package stats

import (
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
)

type Client struct {
	provider string
	statsdc  *statsd.Client
}

var instance *Client

func InitializeClient(provider string) error {
	if instance == nil {
		instance = &Client{}
		if provider == "datadog" {
			statsd, err := statsd.New("127.0.0.1:8125")
			if err != nil {
				return err
			}
			instance.statsdc = statsd
		} else {
			statsd, err := statsd.New(provider)
			if err != nil {
				return err
			}
			instance.statsdc = statsd
		}

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
