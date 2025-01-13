package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// RecoveryMiddleware handles panics and returns a 500 error
func RecoveryMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("Panic recovered: %v", r)
					c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Internal server error"})
				}
			}()

			return next(c)
		}
	}
}
