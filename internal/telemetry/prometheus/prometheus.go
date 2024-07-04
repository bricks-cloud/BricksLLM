package prometheus

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Enabled bool
	Port    string
}

var (
	config           Config
	counterMetrics   = make(map[string]*prometheus.CounterVec)
	histogramMetrics = make(map[string]*prometheus.HistogramVec)
)

func Init(cfg Config) error {
	config = cfg
	initMetrics()
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+cfg.Port, nil)
	return nil
}

func Incr(name string, tags []string, rate float64) {
	counterMetric, exists := counterMetrics[name]
	if !exists {
		return
	}
	counterMetric.WithLabelValues(tags...).Inc()
}

func Timing(name string, value time.Duration, tags []string, rate float64) {
	histogramMetric, exists := histogramMetrics[name]
	if !exists {
		return
	}
	histogramMetric.WithLabelValues(tags...).Observe(float64(value))
}
