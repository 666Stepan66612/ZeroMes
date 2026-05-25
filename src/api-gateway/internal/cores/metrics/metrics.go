package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    // Counter — RPS и ошибки auth
    HttpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    // Histogram — latency
    HttpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
        },
        []string{"method", "endpoint"},
    )

    // Gauge — активные WS соединения
    ActiveWebSocketConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "websocket_active_connections",
            Help: "Number of active WebSocket connections",
        },
    )

    // Counter — ошибки аутентификации по типу
    AuthErrorsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "auth_errors_total",
            Help: "Total number of auth errors",
        },
        []string{"reason"}, // missing_token, invalid_token, revoked_token
    )
)

func Init() {
    prometheus.MustRegister(
        HttpRequestsTotal,
        HttpRequestDuration,
        ActiveWebSocketConnections,
        AuthErrorsTotal,
    )
}