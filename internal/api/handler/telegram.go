package handler

import (
	"net/http"
	"time"

	"caatsm/internal/api/dto"
	"caatsm/internal/iface"
	"caatsm/internal/service"

	"github.com/labstack/echo/v4"
)

type TelegramHandler struct {
	service *service.TelegramService
}

func NewTelegramHandler(svc *service.TelegramService) *TelegramHandler {
	return &TelegramHandler{
		service: svc,
	}
}

// GetTelegram retrieves a telegram by ID
// @Summary Get telegram by ID
// @Tags telegrams
// @Accept json
// @Produce json
// @Param id path string true "Telegram UUID"
// @Success 200 {object} dto.TelegramResponse
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/telegrams/{id} [get]
func (h *TelegramHandler) GetTelegram(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "telegram ID is required")
	}

	msg, err := h.service.GetTelegram(c.Request().Context(), id)
	if err != nil {
		if err.Error() == "telegram not found: "+id {
			return echo.NewHTTPError(http.StatusNotFound, "telegram not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, dto.ToTelegramResponse(msg))
}

// ListTelegrams lists telegrams with filters and pagination
// @Summary List telegrams
// @Tags telegrams
// @Accept json
// @Produce json
// @Param category query string false "Category filter"
// @Param messageId query string false "Message ID filter"
// @Param primaryAddress query string false "Primary address filter"
// @Param priorityIndicator query string false "Priority indicator filter"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Param limit query int false "Limit (default: 100, max: 1000)"
// @Param offset query int false "Offset (default: 0)"
// @Param orderBy query string false "Order by field (default: received_at)"
// @Param orderDirection query string false "Order direction (ASC/DESC, default: DESC)"
// @Success 200 {object} dto.ListTelegramsResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/telegrams [get]
func (h *TelegramHandler) ListTelegrams(c echo.Context) error {
	var req dto.ListTelegramsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request parameters")
	}

	filters := &iface.TelegramFilters{
		Category:          req.Category,
		MessageID:         req.MessageID,
		PrimaryAddress:    req.PrimaryAddress,
		PriorityIndicator: req.PriorityIndicator,
		Limit:             req.Limit,
		Offset:            req.Offset,
		OrderBy:           req.OrderBy,
		OrderDirection:    req.OrderDirection,
	}

	if !req.StartTime.IsZero() {
		filters.StartTime = &req.StartTime
	}
	if !req.EndTime.IsZero() {
		filters.EndTime = &req.EndTime
	}

	messages, total, err := h.service.ListTelegrams(c.Request().Context(), filters)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	responses := make([]*dto.TelegramResponse, len(messages))
	for i, msg := range messages {
		responses[i] = dto.ToTelegramResponse(msg)
	}

	return c.JSON(http.StatusOK, dto.ListTelegramsResponse{
		Telegrams: responses,
		Total:     total,
		Limit:     filters.Limit,
		Offset:    filters.Offset,
	})
}

// GetTelegramsByTimeRange retrieves telegrams within a time range
// @Summary Get telegrams by time range
// @Tags telegrams
// @Accept json
// @Produce json
// @Param start query string true "Start time (RFC3339)"
// @Param end query string true "End time (RFC3339)"
// @Success 200 {array} dto.TelegramResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/telegrams/time-range [get]
func (h *TelegramHandler) GetTelegramsByTimeRange(c echo.Context) error {
	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	if startStr == "" || endStr == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start and end time are required")
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start time format (use RFC3339)")
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end time format (use RFC3339)")
	}

	messages, err := h.service.GetTelegramsByTimeRange(c.Request().Context(), start, end)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	responses := make([]*dto.TelegramResponse, len(messages))
	for i, msg := range messages {
		responses[i] = dto.ToTelegramResponse(msg)
	}

	return c.JSON(http.StatusOK, responses)
}

