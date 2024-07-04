package prometheus

import (
	metricname "github.com/bricks-cloud/bricksllm/internal/telemetry/metric_name"
	"github.com/prometheus/client_golang/prometheus"
)

func initMetrics() {
	{
		metricName := metricname.COUNTER_AUTHENTICATOR_FOUND_KEY_FROM_MEMDB
		counterMetrics[metricName] = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metricName,
			},
			[]string{},
		)
		prometheus.MustRegister(counterMetrics[metricName])
	}
	{
		// TODO add other metrics here
	}
}
