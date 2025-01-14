package middlewares

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func Require2FA() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isExempt2FAEndpoint(c.Path()) {
				return next(c)
			}

			userEmail, ok := c.Get("user_email").(string)
			if !ok || userEmail == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "não autorizado: email do usuário ausente",
				})
			}

			code := c.Request().Header.Get("X-2FA-Code")
			if code == "" {
				return c.JSON(http.StatusForbidden, echo.Map{
					"error": "código 2FA necessário",
				})
			}

			var user models.OAuthUser
			if err := repositories.DBConnection.Where("email = ?", userEmail).First(&user).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"error": "falha ao buscar usuário",
				})
			}

			if user.TwoFASecret == "" {
				return c.JSON(http.StatusForbidden, echo.Map{
					"error": "2FA não configurado para o usuário",
				})
			}

			isValid, err := utils.VerifyTwoFactorCode(user.TwoFASecret, code)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"error": fmt.Sprintf("Erro na validação 2FA: %v", err),
				})
			}
			if !isValid {
				return c.JSON(http.StatusForbidden, echo.Map{
					"error": "código 2FA inválido",
				})
			}

			return next(c)
		}
	}
}

func isExempt2FAEndpoint(path string) bool {
	exemptPaths := []string{
		"/api/auth/2fa/setup",
		"/api/auth/2fa/enable",
		"/api/auth/login",
	}

	for _, exemptPath := range exemptPaths {
		if strings.HasPrefix(path, exemptPath) {
			return true
		}
	}
	return false
}
