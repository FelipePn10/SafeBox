package utils

import (
	"time"

	"github.com/sirupsen/logrus"
)

func RefreshOAuthToken(refreshToken string) (*Tokens, error) {
	logrus.Info("Initiating OAuth token refresh")
	claims, err := ValidateOAuthToken(refreshToken)
	if err != nil {
		logrus.Error("Error validating OAuth token:", err)
		return nil, err
	}

	accessToken, err := generateJWT(claims.Username, 15*time.Minute)
	if err != nil {
		logrus.Error("Error generating new access token: ", err)
		return nil, err
	}

	newRefreshToken, err := generateJWT(claims.Username, 7*24*time.Hour)
	if err != nil {
		logrus.Error("Error generating new refresh token: ", err)
		return nil, err
	}

	logrus.Info("OAuth tokens updated successfully")
	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
