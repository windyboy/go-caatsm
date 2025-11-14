package middleware

import (
	"time"

	applog "caatsm/internal/infra/log"

	"github.com/labstack/echo/v4"
)

// Logger returns a middleware that logs HTTP requests
func Logger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()
			res := c.Response()

			logger := applog.Sugared()
			logger.Infof(
				"method=%s uri=%s status=%d latency=%s",
				req.Method,
				req.RequestURI,
				res.Status,
				time.Since(start),
			)

			return err
		}
	}
}

