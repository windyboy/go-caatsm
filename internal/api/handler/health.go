package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type dependency struct {
	name  string
	check func(ctx context.Context) error
}

// HealthOption configures the health handler dependencies.
type HealthOption func(*HealthHandler)

// HealthHandler reports overall service health and dependency readiness.
type HealthHandler struct {
	dependencies []dependency
	timeout      time.Duration
}

// NewHealthHandler creates a health handler with optional dependency checks.
func NewHealthHandler(opts ...HealthOption) *HealthHandler {
	h := &HealthHandler{
		timeout: 2 * time.Second,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithCheckTimeout overrides the default dependency check timeout.
func WithCheckTimeout(d time.Duration) HealthOption {
	return func(h *HealthHandler) {
		if d > 0 {
			h.timeout = d
		}
	}
}

// WithDatabase registers a database ping dependency.
func WithDatabase(db interface{ Ping(context.Context) error }) HealthOption {
	return func(h *HealthHandler) {
		if db == nil {
			return
		}
		h.dependencies = append(h.dependencies, dependency{
			name: "database",
			check: func(ctx context.Context) error {
				return db.Ping(ctx)
			},
		})
	}
}

// WithRedis registers a redis dependency using the provided client.
func WithRedis(client interface {
	Ping(context.Context) interface{ Err() error }
}) HealthOption {
	return func(h *HealthHandler) {
		if client == nil {
			return
		}
		h.dependencies = append(h.dependencies, dependency{
			name: "redis",
			check: func(ctx context.Context) error {
				res := client.Ping(ctx)
				if res == nil {
					return fmt.Errorf("redis ping returned nil")
				}
				return res.Err()
			},
		})
	}
}

// WithNATS registers a NATS dependency that validates the connection status.
func WithNATS(conn interface{ Status() nats.Status }) HealthOption {
	return func(h *HealthHandler) {
		if conn == nil {
			return
		}
		h.dependencies = append(h.dependencies, dependency{
			name: "nats",
			check: func(ctx context.Context) error {
				status := conn.Status()
				if status != nats.CONNECTED {
					return fmt.Errorf("nats status: %s", status)
				}
				return nil
			},
		})
	}
}

// Health returns basic health status
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Ready returns readiness status
// @Summary Readiness check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/ready [get]
func (h *HealthHandler) Ready(c echo.Context) error {
	if len(h.dependencies) == 0 {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ready",
		})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	status := http.StatusOK
	results := make(map[string]string, len(h.dependencies))

	for _, dep := range h.dependencies {
		if err := dep.check(ctx); err != nil {
			results[dep.name] = err.Error()
			status = http.StatusServiceUnavailable
		} else {
			results[dep.name] = "ok"
		}
	}

	payload := map[string]interface{}{
		"status":       http.StatusText(status),
		"dependencies": results,
	}

	return c.JSON(status, payload)
}

// Live returns liveness status
// @Summary Liveness check
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/live [get]
func (h *HealthHandler) Live(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "alive",
	})
}
