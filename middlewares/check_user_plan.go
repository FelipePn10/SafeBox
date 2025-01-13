package middlewares

import (
	"SafeBox/models"
	"net/http"

	"github.com/labstack/echo/v4"
)

// CheckUserPlanMiddleware checks the user's plan and storage limits
func CheckUserPlanMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*models.User)
			if !ok {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "User not found in context"})
			}

			// Check if the user is on the free plan and if the storage limit has been exceeded
			if user.Plan == "free" && user.StorageUsed >= user.StorageLimit {
				return c.JSON(http.StatusForbidden, map[string]interface{}{"error": "Storage limit exceeded"})
			}

			return next(c)
		}
	}
}
