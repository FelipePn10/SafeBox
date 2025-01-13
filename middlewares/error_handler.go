package middlewares

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func ErrorHandler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				logrus.Error("Error request: ", err)
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to process request"})
			}
			return nil
		}
	}
}
