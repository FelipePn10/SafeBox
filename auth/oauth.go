package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var OAuthConfig *oauth2.Config

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading environment variables: ", err)
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	authURL := os.Getenv("AUTH_URL")
	tokenURL := os.Getenv("TOKEN_URL")

	OAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

func GenerateOAuthToken(username string) (*oauth2.Token, error) {
	logrus.Infof("Generating OAuth token for user: %s", username)
	token := &oauth2.Token{
		AccessToken:  fmt.Sprintf("access-token-%s", username),
		RefreshToken: fmt.Sprintf("refresh-token-%s", username),
	}
	return token, nil
}

func RefreshOAuthToken(refreshToken string) (*oauth2.Token, error) {
	logrus.Infof("Updating OAuth token with refresh token: %s", refreshToken)
	token := &oauth2.Token{RefreshToken: refreshToken}
	newToken, err := OAuthConfig.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	return newToken, nil
}

func RevokeToken(token string) error {
	logrus.Infof("Revoking OAuth token: %s", token)
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
