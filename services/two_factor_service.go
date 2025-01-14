package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
	"errors"
)

var (
	ErrInvalid2FACode = errors.New("código 2FA inválido")
	ErrUserNotFound   = errors.New("usuário não encontrado")
	ErrInvalidInput   = errors.New("entrada inválida")
)

type TwoFactorService struct {
	twoFactorRepo *repositories.TwoFactorRepository
}

func NewTwoFactorService(twoFactorRepo *repositories.TwoFactorRepository) *TwoFactorService {
	return &TwoFactorService{twoFactorRepo: twoFactorRepo}
}

func (s *TwoFactorService) Setup2FA(email string) (string, string, error) {
	if email == "" {
		return "", "", ErrInvalidInput
	}

	secret, qrCodeURL, err := utils.Generate2FASecret(email, utils.DefaultTwoFactorConfig())
	if err != nil {
		return "", "", err
	}

	tempSecret := &models.TempTwoFASecret{
		UserEmail: email,
		Secret:    secret,
	}

	if err := s.twoFactorRepo.SaveTempSecret(tempSecret); err != nil {
		return "", "", err
	}

	return secret, qrCodeURL, nil
}

func (s *TwoFactorService) Enable2FA(email, code string) error {
	if email == "" || code == "" {
		return ErrInvalidInput
	}

	tempSecret, err := s.twoFactorRepo.FindTempSecretByEmail(email)
	if err != nil {
		return err
	}

	isValid, err := utils.VerifyTwoFactorCode(tempSecret.Secret, code)
	if err != nil {
		return err
	}
	if !isValid {
		return ErrInvalid2FACode
	}

	return s.twoFactorRepo.DeleteTempSecret(tempSecret)
}
