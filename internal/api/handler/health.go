package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	// Can add dependencies for health checks (db, cache, etc.)
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
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
	// TODO: Add actual readiness checks (database, cache, etc.)
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ready",
	})
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

