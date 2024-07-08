package prometheus

import (
	metricname "github.com/bricks-cloud/bricksllm/internal/telemetry/metric_name"
	"github.com/prometheus/client_golang/prometheus"
)

func (c *Client) initMetrics() {
	{
		metricName := metricname.COUNTER_AUTHENTICATOR_FOUND_KEY_FROM_MEMDB
		c.CounterMetrics[metricName] = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricName,
			},
			[]string{},
		)
		prometheus.MustRegister(c.CounterMetrics[metricName])
	}
	{
		// TODO add other metrics here
	}
}
