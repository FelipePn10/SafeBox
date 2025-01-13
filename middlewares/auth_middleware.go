package middlewares

import (
	"SafeBox/models"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// AuthConfig holds configuration for authentication middleware
type AuthConfig struct {
	RequireToken      bool
	Require2FA        bool
	RequirePermission []models.Permission
}

// NewAuthMiddleware creates a new authentication middleware with the given configuration
func NewAuthMiddleware(config AuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Token validation
			if config.RequireToken {
				token := c.Request().Header.Get("Authorization")
				if token == "" {
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": "Token de autorização não encontrado",
					})
				}

				userEmail, err := ValidateGoogleToken(c, token)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": fmt.Sprintf("Token inválido: %v", err),
					})
				}

				// Fetch user from database
				var oauthUser models.OAuthUser
				// TODO: Implement user fetch from database using userEmail
				// repositories.DBConnection.Where("email = ?", userEmail).First(&oauthUser)

				// Store user in context for later use
				c.Set("oauth_user", oauthUser)

				// 2FA validation if required
				if config.Require2FA && !validate2FA(userEmail) {
					return c.JSON(http.StatusForbidden, echo.Map{
						"error": "Validação 2FA falhou",
					})
				}
			}

			// Permission validation
			if len(config.RequirePermission) > 0 {
				oauthUser, ok := c.Get("oauth_user").(models.OAuthUser)
				if !ok {
					return c.JSON(http.StatusUnauthorized, echo.Map{
						"error": "Usuário não autenticado",
					})
				}

				for _, permission := range config.RequirePermission {
					if !hasPermission(oauthUser.Permissions, permission) {
						return c.JSON(http.StatusForbidden, echo.Map{
							"error": "Permissão negada",
						})
					}
				}
			}

			return next(c)
		}
	}
}

// ValidateGoogleToken validates the Google OAuth token and returns the user's email
func ValidateGoogleToken(c echo.Context, token string) (string, error) {
	service, err := oauth2.NewService(c.Request().Context(), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return "", fmt.Errorf("falha ao criar serviço OAuth2: %w", err)
	}

	tokenInfoCall := service.Tokeninfo()
	tokenInfoCall.AccessToken(token)
	tokenInfo, err := tokenInfoCall.Do()
	if err != nil {
		return "", fmt.Errorf("falha ao validar token: %w", err)
	}

	if tokenInfo.Email == "" || !tokenInfo.VerifiedEmail {
		return "", fmt.Errorf("email não verificado ou ausente")
	}

	return tokenInfo.Email, nil
}

// validate2FA verifies 2FA for a given user
func validate2FA(email string) bool {
	// TODO: Implement actual 2FA validation logic
	return true
}

// hasPermission checks if a permission exists within a slice of permissions
func hasPermission(permissions []models.Permission, target models.Permission) bool {
	for _, permission := range permissions {
		if permission == target {
			return true
		}
	}
	return false
}

// Convenience functions for common middleware configurations
func RequireAuth() echo.MiddlewareFunc {
	return NewAuthMiddleware(AuthConfig{
		RequireToken: true,
	})
}

func RequireAuthWith2FA() echo.MiddlewareFunc {
	return NewAuthMiddleware(AuthConfig{
		RequireToken: true,
		Require2FA:   true,
	})
}

func RequirePermissions(permissions ...models.Permission) echo.MiddlewareFunc {
	return NewAuthMiddleware(AuthConfig{
		RequireToken:      true,
		RequirePermission: permissions,
	})
}
