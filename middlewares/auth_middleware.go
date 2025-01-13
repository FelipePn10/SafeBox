package middlewares

import (
	"SafeBox/models"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// AuthMiddleware authenticates users via Google OAuth token, validates 2FA, and checks permissions.
func AuthMiddleware(permissions ...models.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Request().Header.Get("Authorization")
			if token == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": "missing authorization token",
				})
			}
			userEmail, err := ValidateGoogleToken(c, token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{
					"error": fmt.Sprintf("invalid token: %v", err),
				})
			}

			// Verifica 2FA (implementação mockada)
			if !validate2FA(userEmail) {
				return c.JSON(http.StatusForbidden, echo.Map{
					"error": "2FA validation failed",
				})
			}
			userPermissions := getUserPermissions(userEmail) // Implementação real aqui

			for _, permission := range permissions {
				if !hasPermission(userPermissions, permission) {
					return c.JSON(http.StatusForbidden, echo.Map{
						"error": "permission denied",
					})
				}
			}

			return next(c)
		}
	}
}

// ValidateGoogleToken validates the Google OAuth token and returns the user's email.
func ValidateGoogleToken(c echo.Context, token string) (string, error) {
	service, err := oauth2.NewService(c.Request().Context(), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return "", fmt.Errorf("failed to create OAuth2 service: %w", err)
	}

	tokenInfoCall := service.Tokeninfo()
	tokenInfoCall.AccessToken(token)
	tokenInfo, err := tokenInfoCall.Do()
	if err != nil {
		return "", fmt.Errorf("failed to validate token: %w", err)
	}

	if tokenInfo.Email == "" || !tokenInfo.VerifiedEmail {
		return "", fmt.Errorf("email not verified or missing")
	}

	return tokenInfo.Email, nil
}

// validate2FA is a placeholder function for 2FA validation logic.
func validate2FA(email string) bool {
	// Mock validation for demonstration purposes.
	// Replace with actual 2FA logic (e.g., querying a database or verifying OTPs).
	return true
}

// hasPermission checks if a user has a specific permission.
func hasPermission(permissions []models.Permission, target models.Permission) bool {
	for _, permission := range permissions {
		if permission == target {
			return true
		}
	}
	return false
}

// getUserPermissions retrieves user permissions (mock implementation).
func getUserPermissions(email string) []models.Permission {
	// Replace with actual logic to fetch user permissions from a database or another source.
	return []models.Permission{
		models.Permission("read"),
		models.Permission("write"),
	}
}
