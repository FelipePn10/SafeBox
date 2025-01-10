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
	// Definir plano e limite de armazenamento para novos usuários
	user.Plan = "free"
	user.StorageLimit = 1024 * 1024 * 1024 * 1024 // 1024 GB

	if err := s.userRepo.Create(user); err != nil {
		logrus.Error("Erro ao registrar usuário: ", err)
		return errors.New("erro ao registrar usuário")
	}
	return nil
}

func (s *AuthService) Login(username, password string) (string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		logrus.Error("Usuário não encontrado: ", err)
		return "", errors.New("usuário não encontrado")
	}

	if !utils.ComparePasswords(user.Password, password) {
		logrus.Error("Senha inválida")
		return "", errors.New("senha inválida")
	}

	tokens, err := utils.GenerateOAuthToken(user.Username)
	if err != nil {
		logrus.Error("Erro ao gerar token: ", err)
		return "", errors.New("erro ao gerar token")
	}

	return tokens.AccessToken, nil
}
