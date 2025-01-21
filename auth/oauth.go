package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"SafeBox/models"
	"SafeBox/repositories"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	GoogleOAuthConfig *oauth2.Config
	stateTokens       = make(map[string]bool)
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading environment variables: ", err)
	}

	redirectURL := os.Getenv("OAUTH_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/oauth/callback"
	}

	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

type OAuthHandler struct {
	userRepo repositories.UserRepository
}

func NewOAuthHandler(userRepo repositories.UserRepository) *OAuthHandler {
	return &OAuthHandler{userRepo: userRepo}
}

func (h *OAuthHandler) HandleLogin(c echo.Context) error {
	logger := logrus.WithField("handler", "HandleLogin")
	logger.Info("Iniciando fluxo de login OAuth")

	state := generateStateToken()
	if state == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate state token"})
	}

	url := GoogleOAuthConfig.AuthCodeURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *OAuthHandler) HandleCallback(c echo.Context) error {
	logger := logrus.WithField("handler", "HandleCallback")
	logger.Info("Processando callback OAuth")

	state := c.QueryParam("state")
	if !stateTokens[state] {
		logger.Error("Invalid state token received")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid state token"})
	}
	delete(stateTokens, state)

	code := c.QueryParam("code")
	if code == "" {
		logger.Error("No authorization code received")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Authorization code not provided"})
	}

	token, err := GoogleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		logger.WithError(err).Error("Failed to exchange authorization code for token")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to exchange token"})
	}

	userInfo, err := getUserInfo(token)
	if err != nil {
		logger.WithError(err).Error("Failed to get user info from Google")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get user info"})
	}

	user := &models.OAuthUser{
		Email:        userInfo["email"].(string),
		Username:     userInfo["name"].(string),
		Avatar:       userInfo["picture"].(string),
		Provider:     "google",
		StorageLimit: 20 * 1024 * 1024 * 1024, // 20 GB
		Plan:         "free",
	}

	if err := h.userRepo.CreateOrUpdate(user); err != nil {
		logger.WithError(err).Error("Failed to create/update user in database")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to process user"})
	}

	return c.JSON(http.StatusOK, user)
}

func getUserInfo(token *oauth2.Token) (map[string]interface{}, error) {
	client := GoogleOAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch user info: status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

func generateStateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		logrus.WithError(err).Error("Failed to generate state token")
		return ""
	}
	token := base64.StdEncoding.EncodeToString(b)
	stateTokens[token] = true
	return token
}

func RevokeToken(token string) error {
	logger := logrus.WithField("handler", "RevokeToken")
	logger.Info("Revogando token OAuth")

	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/revoke",
		strings.NewReader(fmt.Sprintf("token=%s", token)))
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
