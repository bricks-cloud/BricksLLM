package proxy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func getOtelMiddlware() gin.HandlerFunc {
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

func getOtelTransport() *otelhttp.Transport {
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
