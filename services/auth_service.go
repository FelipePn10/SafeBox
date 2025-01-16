package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
)

type AuthService struct {
	userRepo repositories.UserRepository // Remove the pointer
}

func NewAuthService(userRepo repositories.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(user *models.OAuthUser) error {
	return s.userRepo.CreateUser(user)
}

func (s *AuthService) ValidateToken(token string) (*utils.TokenClaims, error) {
	return utils.ValidateToken(token)
}
