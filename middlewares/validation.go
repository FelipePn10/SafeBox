package middlewares

import (
	"SafeBox/models"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get("user").(*models.OAuthUser) // Assume que o usuário foi definido no contexto após OAuth
			if user == nil {
				return c.JSON(http.StatusUnauthorized, "Unauthorized")
			}
			return next(c)
		}
	}
}
func extractToken(c echo.Context) string {
	token := c.Request().Header.Get("Authorization")
	return strings.TrimPrefix(token, "Bearer ")
}

func ValidateOAuthToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Request().Header.Get("Authorization")
		if token == "" {
			return c.JSON(http.StatusUnauthorized, "Missing token")
		}

		// Valida o token com o Google
		_, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s", token))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, "Invalid token")
		}

		return next(c)
	}
}
