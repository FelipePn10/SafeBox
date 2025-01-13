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
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading environment variables: ", err)
	}

	redirectURL := os.Getenv("OAUTH_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/oauth/callback" // Default fallback
	}

	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// GenerateOAuthToken creates a token for the given username
func GenerateOAuthToken(username string) (*oauth2.Token, error) {
	logrus.WithField("username", username).Info("Generating OAuth token")
	return &oauth2.Token{
		AccessToken:  fmt.Sprintf("access-token-%s", username),
		RefreshToken: fmt.Sprintf("refresh-token-%s", username),
	}, nil
}

// RefreshOAuthToken refreshes an existing OAuth token
func RefreshOAuthToken(refreshToken string) (*oauth2.Token, error) {
	logrus.WithField("refresh_token", refreshToken).Info("Refreshing OAuth token")
	token := &oauth2.Token{RefreshToken: refreshToken}
	newToken, err := GoogleOAuthConfig.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	return newToken, nil
}

// RevokeToken revokes the given OAuth token
func RevokeToken(token string) error {
	logrus.WithField("token", token).Info("Revoking OAuth token")
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

// generateStateToken generates a secure, random state token for OAuth flow
func generateStateToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate state token")
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// OAuthLogin redirects the user to Google's OAuth 2.0 authentication page
func OAuthLogin(c echo.Context) error {
	logrus.Info("Initiating Google OAuth login flow")
	state := generateStateToken()
	url := GoogleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// OAuthCallback handles the OAuth 2.0 callback and retrieves user information
func OAuthCallback(c echo.Context) error {
	logger := logrus.WithField("handler", "OAuthCallback")
	logger.Info("Processing OAuth callback")

	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Authorization code not provided"})
	}

	// TODO: Verify state parameter to prevent CSRF attacks

	ctx := context.Background()
	token, err := GoogleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		logger.WithError(err).Error("Failed to exchange authorization code")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Authentication failed"})
	}

	userInfo, err := GetUserInfoFromGoogle(token)
	if err != nil {
		logger.WithError(err).Error("Failed to retrieve user info")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve user information"})
	}

	oauthUser := models.OAuthUser{
		ID:       fmt.Sprintf("%v", userInfo["sub"]),
		Email:    fmt.Sprintf("%v", userInfo["email"]),
		Username: fmt.Sprintf("%v", userInfo["name"]),
		Avatar:   fmt.Sprintf("%v", userInfo["picture"]),
	}

	result := repositories.DBConection.Create(&oauthUser)
	if result.Error != nil {
		logger.WithError(result.Error).Error("Failed to create user in database")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
	}

	return c.JSON(http.StatusOK, oauthUser)
}

// GetUserInfoFromGoogle retrieves user information from Google's UserInfo API
func GetUserInfoFromGoogle(token *oauth2.Token) (map[string]interface{}, error) {
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
