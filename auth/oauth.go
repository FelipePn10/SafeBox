package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var OAuthConfig = &oauth2.Config{
	ClientID:     "l8iZ8VXHsSPXEdw6vcQdHxYrwq4u6czI",
	ClientSecret: "vK_67EkC-IT0Tc63d7Kt7p_ObEff9TNn6bh7iy8nQXU9Yl5Q3gvXzUG06JQxicsV",
	Scopes:       []string{"openid", "profile", "email"},
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://accounts.google.com/o/oauth2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
	},
}

func GenerateOAuthToken(username string) (*oauth2.Token, error) {
	logrus.Infof("Gerando token OAuth para o usuário: %s", username)
	token := &oauth2.Token{
		AccessToken:  fmt.Sprintf("access-token-%s", username),
		RefreshToken: fmt.Sprintf("refresh-token-%s", username),
	}
	return token, nil
}

func RefreshOAuthToken(refreshToken string) (*oauth2.Token, error) {
	logrus.Infof("Atualizando token OAuth com refresh token: %s", refreshToken)
	token := &oauth2.Token{RefreshToken: refreshToken}
	newToken, err := OAuthConfig.TokenSource(context.Background(), token).Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	return newToken, nil
}

func RevokeToken(token string) error {
	logrus.Infof("Revogando token OAuth: %s", token)
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
