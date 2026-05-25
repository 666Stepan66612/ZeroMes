package middleware

import (
    "api-gateway/internal/cores/metrics"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start).Seconds()
        status := strconv.Itoa(c.Writer.Status())
        endpoint := c.FullPath() // "/auth/register", "/ws" и т.д.
        if endpoint == "" {
            endpoint = "unknown"
        }

        metrics.HttpRequestsTotal.
            WithLabelValues(c.Request.Method, endpoint, status).
            Inc()

        metrics.HttpRequestDuration.
            WithLabelValues(c.Request.Method, endpoint).
            Observe(duration)
    }
}