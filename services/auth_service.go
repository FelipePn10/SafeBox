package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
	"errors"

	"github.com/sirupsen/logrus"
)

type AuthService struct {
	userRepo *repositories.UserRepository
}

func NewAuthService(userRepo *repositories.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(user *models.User) error {
	// Set storage plan and limit for new users
	user.Plan = "free"
	user.StorageLimit = 1024 * 1024 * 1024 * 1024 // 1024 GB

	if err := s.userRepo.Create(user); err != nil {
		logrus.Error("Error registering user: ", err)
		return errors.New("error registering user")
	}
	return nil
}

func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		logrus.Error("User not found: ", err)
		return "", errors.New("user not found")
	}

	if !utils.ComparePasswords(user.Password, password) {
		logrus.Error("Invalid password")
		return "", errors.New("invalid password")
	}

	tokens, err := utils.GenerateOAuthToken(user.Username)
	if err != nil {
		logrus.Error("Error generating tokens: ", err)
		return "", errors.New("error generating token")
	}

	return tokens.AccessToken, nil
}
