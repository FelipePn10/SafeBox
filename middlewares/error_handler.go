package middlewares

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type ErrorResponse struct {
	Error     string      `json:"error"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

func ErrorHandler() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			logEntry := logrus.WithFields(logrus.Fields{
				"method":    c.Request().Method,
				"path":      c.Path(),
				"status":    c.Response().Status,
				"client_ip": c.RealIP(),
			})

			// Tratamento de erros específicos
			var httpErr *echo.HTTPError
			if errors.As(err, &httpErr) {
				logEntry.WithField("code", httpErr.Code).Error(httpErr.Message)
				return c.JSON(httpErr.Code, ErrorResponse{
					Error:     http.Error(httpErr.Message, httpErr.Code),
					RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				})
			}

			// Erro genérico
			logEntry.WithError(err).Error("Unexpected error")
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:     "internal_server_error",
				Details:   err.Error(),
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			})
		}
	}
}
