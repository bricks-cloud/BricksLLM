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

type Client struct {
	Config           Config
	CounterMetrics   map[string]*prometheus.CounterVec
	HistogramMetrics map[string]*prometheus.HistogramVec
}

func Init(cfg Config) (*Client, error) {
	c := &Client{
		Config:           cfg,
		CounterMetrics:   make(map[string]*prometheus.CounterVec),
		HistogramMetrics: make(map[string]*prometheus.HistogramVec),
	}

	c.initMetrics()

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":"+cfg.Port, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) Incr(name string, tags []string, rate float64) {
	if c == nil {
		return
	}

	counterMetric, exists := c.CounterMetrics[name]
	if !exists {
		return
	}

	counterMetric.WithLabelValues(tags...).Inc()
}

func (c *Client) Timing(name string, value time.Duration, tags []string, rate float64) {
	if c == nil {
		return
	}

	histogramMetric, exists := c.HistogramMetrics[name]
	if !exists {
		return
	}

	histogramMetric.WithLabelValues(tags...).Observe(float64(value))
}
