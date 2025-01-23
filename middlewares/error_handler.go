package middlewares

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func ErrorHandler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			logrus.WithFields(logrus.Fields{
				"error": err.Error(),
				"path":  c.Path(),
			}).Error("Request error")

			var httpErr *echo.HTTPError
			if errors.As(err, &httpErr) {
				return c.JSON(httpErr.Code, ErrorResponse{
					Error:   http.StatusText(httpErr.Code),
					Message: fmt.Sprintf("%v", httpErr.Message),
				})
			}

			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error: "internal_server_error",
			})
		}
	}
}
