package api

import (
	"caatsm/internal/api/handler"
	"caatsm/internal/api/middleware"
	"caatsm/internal/service"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter configures and returns the Echo router.
func SetupRouter(telegramService *service.TelegramService, healthOptions ...handler.HealthOption) *echo.Echo {
	e := echo.New()

	// Middleware
	e.Use(middleware.Recovery())
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Use(middleware.Metrics())
	e.Use(echoMiddleware.RequestID())

	// Health check endpoints
	healthHandler := handler.NewHealthHandler(healthOptions...)
	e.GET("/health", healthHandler.Health)
	e.GET("/health/ready", healthHandler.Ready)
	e.GET("/health/live", healthHandler.Live)

	// Prometheus metrics endpoint
	e.GET("/metrics", func(c echo.Context) error {
		promhttp.Handler().ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// API v1 routes
	v1 := e.Group("/api/v1")
	{
		telegramHandler := handler.NewTelegramHandler(telegramService)
		v1.GET("/telegrams", telegramHandler.ListTelegrams)
		v1.GET("/telegrams/:id", telegramHandler.GetTelegram)
		v1.GET("/telegrams/time-range", telegramHandler.GetTelegramsByTimeRange)
	}

	return e
}

