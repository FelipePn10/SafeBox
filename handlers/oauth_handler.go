package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"SafeBox/models"
	"SafeBox/repositories"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthHandler struct {
	config     *oauth2.Config
	userRepo   repositories.UserRepository
	stateStore *sync.Map
}

func NewOAuthHandler(userRepo repositories.UserRepository) (*OAuthHandler, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning loading .env: %v", err)
	}

	config := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  getRedirectURL(),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &OAuthHandler{
		config:     config,
		userRepo:   userRepo,
		stateStore: &sync.Map{},
	}, nil
}

func getRedirectURL() string {
	if url := os.Getenv("OAUTH_REDIRECT_URL"); url != "" {
		return url
	}
	return "http://localhost:8080/oauth/callback"
}

func (h *OAuthHandler) Login(c echo.Context) error {
	state := generateStateToken()
	h.stateStore.Store(state, time.Now().Add(5*time.Minute))

	url := h.config.AuthCodeURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *OAuthHandler) Callback(c echo.Context) error {
	ctx := c.Request().Context()
	logger := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"handler":  "OAuthCallback",
		"provider": "google",
	})

	// Validate state token
	state := c.QueryParam("state")
	if !h.validStateToken(state) {
		logger.Warn("Invalid state token")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_state"})
	}

	// Exchange code for token
	code := c.QueryParam("code")
	token, err := h.config.Exchange(ctx, code)
	if err != nil {
		logger.WithError(err).Error("Token exchange failed")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "token_exchange_failed"})
	}

	// Get user info
	userInfo, err := h.getUserInfo(ctx, token)
	if err != nil {
		logger.WithError(err).Error("Failed to get user info")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user_info_fetch_failed"})
	}

	// Create/update user
	user := h.mapUserInfo(userInfo)
	if err := h.userRepo.CreateOrUpdate(user); err != nil {
		logger.WithFields(logrus.Fields{
			"email": user.Email,
			"error": err.Error(),
		}).Error("User creation/update failed")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "user_management_failed"})
	}

	// Set user in context
	c.Set("user", user)

	return c.JSON(http.StatusOK, user)
}

func (h *OAuthHandler) validStateToken(token string) bool {
	expiryTime, ok := h.stateStore.Load(token)
	if !ok {
		return false
	}
	h.stateStore.Delete(token)
	return time.Now().Before(expiryTime.(time.Time))
}

func (h *OAuthHandler) getUserInfo(ctx context.Context, token *oauth2.Token) (map[string]interface{}, error) {
	client := h.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

func (h *OAuthHandler) mapUserInfo(info map[string]interface{}) *models.OAuthUser {
	return &models.OAuthUser{
		Email:        info["email"].(string),
		Username:     info["name"].(string),
		Avatar:       info["picture"].(string),
		Provider:     "google",
		StorageLimit: 20 * 1024 * 1024 * 1024, // 20GB
		Plan:         "free",
	}
}

func generateStateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		logrus.WithError(err).Error("Failed to generate state token")
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
