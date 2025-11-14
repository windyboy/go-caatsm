package middleware

import (
	"caatsm/internal/api/metrics"
	"time"

	"github.com/labstack/echo/v4"
)

// Metrics returns a middleware that records Prometheus metrics
func Metrics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			method := c.Request().Method
			path := c.Path()
			status := c.Response().Status

			metrics.HTTPRequestsTotal.WithLabelValues(method, path, statusCodeToString(status)).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

			return err
		}
	}
}

func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

