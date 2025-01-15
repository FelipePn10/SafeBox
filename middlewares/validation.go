package middlewares

import (
	"SafeBox/utils"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractToken(c)
			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Token de autorização não encontrado")
			}

			// Passa o contexto para ValidateOAuthToken
			claims, err := utils.ValidateOAuthToken(c.Request().Context(), token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Token inválido: %v", err))
			}

			c.Set("user", claims.Username)
			return next(c)
		}
	}
}
