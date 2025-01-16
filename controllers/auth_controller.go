// controllers/auth_controller.go
package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthController struct {
	config      *oauth2.Config
	stateTokens map[string]bool
	userRepo    repositories.UserRepository // Changed from *repositories.UserRepository
}

// NewOAuthController creates a new instance of OAuthController
func NewOAuthController(userRepo repositories.UserRepository) (*OAuthController, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("OAUTH_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, fmt.Errorf("missing required OAuth configuration")
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthController{
		config:      config,
		stateTokens: make(map[string]bool),
		userRepo:    userRepo,
	}, nil
}

// generateStateToken creates a secure random state token for OAuth flow
func (c *OAuthController) generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(b)
	c.stateTokens[token] = true

	// Clean up old tokens after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		delete(c.stateTokens, token)
	}()

	return token, nil
}

// HandleLogin initiates the OAuth login process
func (c *OAuthController) HandleLogin(ctx echo.Context) error {
	state, err := c.generateStateToken()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate state token",
		})
	}

	url := c.config.AuthCodeURL(state)
	return ctx.Redirect(http.StatusTemporaryRedirect, url)
}

// HandleCallback processes the OAuth callback
func (c *OAuthController) HandleCallback(ctx echo.Context) error {
	logger := logrus.WithFields(logrus.Fields{
		"handler": "HandleCallback",
		"method":  "OAuth2",
	})

	state := ctx.QueryParam("state")
	if !c.stateTokens[state] {
		logger.Warn("Invalid state token received")
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid state token",
		})
	}
	delete(c.stateTokens, state)

	code := ctx.QueryParam("code")
	if code == "" {
		logger.Warn("No authorization code received")
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Authorization code not provided",
		})
	}

	token, err := c.config.Exchange(context.Background(), code)
	if err != nil {
		logger.WithError(err).Error("Failed to exchange authorization code")
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to authenticate with provider",
		})
	}

	userInfo, err := c.getUserInfo(token)
	if err != nil {
		logger.WithError(err).Error("Failed to get user info")
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get user information",
		})
	}

	user := &models.OAuthUser{
		Email:        userInfo["email"].(string),
		Username:     userInfo["name"].(string),
		Avatar:       userInfo["picture"].(string),
		Provider:     "google",
		StorageLimit: 5 * 1024 * 1024 * 1024, // 5GB default limit
		Plan:         "free",
	}

	if err := c.userRepo.CreateOrUpdate(user); err != nil {
		logger.WithError(err).Error("Failed to create/update user")
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to process user data",
		})
	}

	// Generate tokens using the utility function
	tokens, err := utils.GenerateOAuthToken(ctx.Request().Context(), user.Username)
	if err != nil {
		logger.WithError(err).Error("Failed to generate tokens")
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate authentication tokens",
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"user":          user,
	})
}

// getUserInfo fetches the user information from Google's userinfo endpoint
func (c *OAuthController) getUserInfo(token *oauth2.Token) (map[string]interface{}, error) {
	client := c.config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}
