package utils

import (
	"time"

	"github.com/sirupsen/logrus"
)

func RefreshOAuthToken(refreshToken string) (*Tokens, error) {
	logrus.Info("Iniciando refresh do token OAuth")
	claims, err := ValidateOAuthToken(refreshToken)
	if err != nil {
		logrus.Error("Erro ao validar o token OAuth: ", err)
		return nil, err
	}

	accessToken, err := generateJWT(claims.Username, 15*time.Minute)
	if err != nil {
		logrus.Error("Erro ao gerar novo access token: ", err)
		return nil, err
	}

	newRefreshToken, err := generateJWT(claims.Username, 7*24*time.Hour)
	if err != nil {
		logrus.Error("Erro ao gerar novo refresh token: ", err)
		return nil, err
	}

	logrus.Info("Tokens OAuth atualizados com sucesso")
	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
