package middlewares

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// AuthMiddleware authenticates users via Google OAuth token and validates their 2FA.
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Request().Header.Get("Authorization")
		if token == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error": "missing authorization token",
			})
		}

		// Validate Google OAuth token
		userEmail, err := ValidateGoogleToken(c, token)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error": fmt.Sprintf("invalid token: %v", err),
			})
		}

		// Check 2FA (mock implementation for now)
		if !validate2FA(userEmail) {
			return c.JSON(http.StatusForbidden, echo.Map{
				"error": "2FA validation failed",
			})
		}

		return next(c)
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
