// Package handlers gerencia as requisições HTTP relacionadas a cotas
package handlers

import (
	"SafeBox/services"
	"net/http"

	"github.com/labstack/echo/v4"
)

type QuotaHandler struct {
	quotaService services.QuotaServiceInterface
}

func NewQuotaHandler(qs services.QuotaServiceInterface) *QuotaHandler {
	return &QuotaHandler{quotaService: qs}
}

func (h *QuotaHandler) GetQuotaUsage(c echo.Context) error {
	userID := c.Get("userID").(uint)
	used, err := h.quotaService.GetCurrentUsage(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"used":  used,
		"limit": h.quotaService.GetLimit(userID),
	})
}
