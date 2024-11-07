package proxy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func getOtelMiddlware(enableOtel bool) gin.HandlerFunc {
	// returns a no-op middleware if OpenTelemetry is disabled
	if !enableOtel {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	spanName := func(r *http.Request) string {
		return "HTTP " + r.Method + " " + r.URL.Path
	}

	md := otelgin.Middleware(
		"bricksllm-proxy",
		otelgin.WithSpanNameFormatter(spanName),
		otelgin.WithPropagators(otel.GetTextMapPropagator()),
		otelgin.WithTracerProvider(otel.GetTracerProvider()),
	)
	return md
}

func getOtelTransport(enableOtel bool) http.RoundTripper {
	// returns a no-op transport if OpenTelemetry is disabled
	if !enableOtel {
		return http.DefaultTransport
	}

	spanName := func(_ string, r *http.Request) string {
		return "HTTP " + r.Method + " " + r.URL.Path
	}
	rt := otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithSpanNameFormatter(spanName),
		otelhttp.WithServerName("bricksllm-proxy"),
	)
	return rt
}
