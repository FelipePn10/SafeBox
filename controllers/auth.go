// auth.go

package controllers

import (
	"SafeBox/auth"
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/services"
	"SafeBox/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	authService *services.AuthService
}

// NewAuthController creates a new instance of AuthController
func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// Register handles user registration
func (a *AuthController) Register(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid request"})
	}

	db := repositories.DBConection
	if err := services.NewAuthService(repositories.NewUserRepository(db)).Register(&user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"message": "User created"})
}

// Login handles user login
func (a *AuthController) Login(c echo.Context) error {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind(&credentials); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error binding JSON")
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid request"})
	}

	logrus.WithFields(logrus.Fields{
		"username": credentials.Username,
	}).Info("User login")

	var user models.User
	result := repositories.DBConection.Where("username = ?", credentials.Username).First(&user)
	if result.Error != nil {
		logrus.WithFields(logrus.Fields{
			"username": credentials.Username,
			"error":    result.Error,
		}).Error("User not found")
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "User not found"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		logrus.WithFields(logrus.Fields{
			"username": credentials.Username,
			"error":    err,
		}).Error("Invalid password")
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Invalid password"})
	}

	token, err := utils.GenerateOAuthToken(user.Username)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username": credentials.Username,
			"error":    err,
		}).Error("Error generating token")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error generating token"})
	}

	logrus.WithFields(logrus.Fields{
		"username": credentials.Username,
	}).Info("User logged in")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expires_in":    15 * 60,
	})
}

// Logout handles user logout
func (a *AuthController) Logout(c echo.Context) error {
	// Get header token
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Authorization token not provided"})
	}

	// Extract the Bearer token
	parts := strings.Split(token, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid Authorization header format"})
	}
	accessToken := parts[1]

	// Revoke the token
	err := auth.RevokeToken(accessToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("Failed to revoke token: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "User logged out successfully"})
}

// RevokeGoogleToken revokes the provided token using Google's API
func RevokeGoogleToken(token string) error {
	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/revoke", strings.NewReader("token="+token))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute revoke request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to revoke token: status %d", resp.StatusCode)
	}

	return nil
}

// RefreshToken handles token refresh
func (a *AuthController) RefreshToken(c echo.Context) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid request"})
	}

	// Refresh the token
	token, err := utils.RefreshOAuthToken(body.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to refresh token"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expires_in":    15 * 60,
	})
}

// AuthMiddleware validates the OAuth token
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenString := c.Request().Header.Get("Authorization")
			if tokenString == "" {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Unauthorized"})
			}

			// Extract the Bearer token
			parts := strings.Split(tokenString, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid Authorization header format"})
			}
			accessToken := parts[1]

			// Validate the token
			claims, err := utils.ValidateOAuthToken(accessToken)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Unauthorized"})
			}

			// Set the username in the context for use by subsequent handlers
			c.Set("username", claims.Username)
			return next(c)
		}
	}
}
