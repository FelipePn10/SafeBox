package middlewares

import (
	"SafeBox/utils"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ValidateTokenMiddleware validates the OAuth token
func ValidateTokenMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Token não fornecido"})
			}

			_, err := utils.ValidateOAuthToken(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Token inválido"})
			}

			return next(c)
		}
	}
}
