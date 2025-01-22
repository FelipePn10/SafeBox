package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"context"
	"fmt"
)

type AuthService struct {
	userRepo    repositories.UserRepository
	oauthConfig *oauth2.Config
}

func NewAuthService(userRepo repositories.UserRepository, oauthConfig *oauth2.Config) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		oauthConfig: oauthConfig,
	}
}

func (s *AuthService) FindOrCreateUser(ctx context.Context, userInfo map[string]interface{}) (*models.OAuthUser, error) {
	email := userInfo["email"].(string)
	user, err := s.userRepo.FindByEmail(email)
	if err == nil {
		return user, nil
	}

	newUser := &models.OAuthUser{
		Email:    email,
		Username: userInfo["name"].(string),
		Avatar:   userInfo["picture"].(string),
		Provider: "google",
	}

	if err := s.userRepo.CreateOrUpdate(newUser); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

func (s *AuthService) ValidateAccessToken(ctx context.Context, token string) (map[string]interface{}, error) {
	client := s.oauthConfig.Client(ctx, &oauth2.Token{AccessToken: token})
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token status: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}
