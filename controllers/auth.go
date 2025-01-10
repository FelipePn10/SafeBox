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

	"github.com/gin-gonic/gin"
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
func (a *AuthController) Register(c *gin.Context) {
	var user models.User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := repositories.DBConection
	if err := services.NewAuthService(repositories.NewUserRepository(db)).Register(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created"})
}

// Login handles user login
func (a *AuthController) Login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&credentials); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Error binding JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
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
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		logrus.WithFields(logrus.Fields{
			"username": credentials.Username,
			"error":    err,
		}).Error("Invalid password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	token, err := utils.GenerateOAuthToken(user.Username)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username": credentials.Username,
			"error":    err,
		}).Error("Error generating token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"username": credentials.Username,
	}).Info("User logged in")

	c.JSON(http.StatusOK, gin.H{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expires_in":    15 * 60,
	})
}

// Logout handles user logout
func (a *AuthController) Logout(c *gin.Context) {
	// Get header token
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization token not provided"})
		return
	}

	// Extract the Bearer token
	parts := strings.Split(token, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Authorization header format"})
		return
	}
	accessToken := parts[1]

	// Revoke the token
	err := auth.RevokeToken(accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to revoke token: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
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
func (a *AuthController) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Renova o Access Token usando o Refresh Token
	token, err := utils.RefreshOAuthToken(body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"expires_in":    15 * 60,
	})
}

// AuthMiddleware validates the OAuth token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Extract the Bearer token
		parts := strings.Split(tokenString, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Authorization header format"})
			c.Abort()
			return
		}
		accessToken := parts[1]

		// Validate the token
		claims, err := utils.ValidateOAuthToken(accessToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}
