package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
	"errors"
)

type AuthService struct {
	userRepo *repositories.UserRepository
}

func NewAuthService(userRepo *repositories.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(user *models.User) error {
	return s.userRepo.CreateUser(user)
}

func (s *AuthService) Login(username, password string) (*models.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, err
	}

	if err := utils.ComparePassword(user.Password, password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

func (s *AuthService) ValidateToken(token string) (*utils.TokenClaims, error) {
	return utils.ValidateToken(token)
}
