package controllers

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig = oauth2.Config{
		ClientID:     "l8iZ8VXHsSPXEdw6vcQdHxYrwq4u6czI",
		ClientSecret: "vK_67EkC-IT0Tc63d7Kt7p_ObEff9TNn6bh7iy8nQXU9Yl5Q3gvXzUG06JQxicsV",
		RedirectURL:  "http://localhost:8080/oauth/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
)

// OAuthLogin redirects the user to Google's OAuth 2.0 authentication page
func OAuthLogin(c *gin.Context) {
	logrus.Info("Redirecting to Google OAuth 2.0 authentication page")
	url := googleOAuthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusFound, url)
}

// OAuthCallback handles the OAuth 2.0 callback and retrieves user information
func OAuthCallback(c *gin.Context) {
	logrus.Info("Receiving callback from Google OAuth 2.0")
	// Retrieve the authorization code from query parameters
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not provided"})
		return
	}

	// Exchange the code for an access token
	ctx := context.Background()
	token, err := googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange authorization code: " + err.Error()})
		return
	}

	// Retrieve user information using the access token
	userInfo, err := GetUserInfoFromGoogle(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user info: " + err.Error()})
		return
	}

	// Map user info to the OAuthUser model
	oauthUser := models.OAuthUser{
		ID:       fmt.Sprintf("%v", userInfo["sub"]),
		Email:    fmt.Sprintf("%v", userInfo["email"]),
		Username: fmt.Sprintf("%v", userInfo["name"]),
		Avatar:   fmt.Sprintf("%v", userInfo["picture"]),
	}
	repositories.DBConection.Create(&oauthUser)

	// Respond with the retrieved user information
	c.JSON(http.StatusOK, oauthUser)
}

// GetUserInfoFromGoogle retrieves user information from Google's UserInfo API
func GetUserInfoFromGoogle(token *oauth2.Token) (map[string]interface{}, error) {
	client := googleOAuthConfig.Client(context.Background(), token)
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
