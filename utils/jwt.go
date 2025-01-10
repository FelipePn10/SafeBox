package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
)

var jwtSecret = []byte("your-secret-key")

// Claims define os dados do token JWT
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateOAuthToken generates an access token and a refresh token
func GenerateOAuthToken(username string) (*Tokens, error) {
	logrus.Infof("Gerando tokens OAuth para o usuário: %s", username)
	// Generate Access Token
	accessToken, err := generateJWT(username, 15*time.Minute) // Valid for 15 minutes
	if err != nil {
		return nil, err
	}

	// Gerar Refresh Token
	refreshToken, err := generateJWT(username, 7*24*time.Hour) // Valid for 7 days
	if err != nil {
		return nil, err
	}

	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateOAuthToken validates a JWT token and returns user data
func ValidateOAuthToken(tokenString string) (*Claims, error) {
	logrus.Info("Validando token OAuth")
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// generateJWT creates a JWT token with expiration time
func generateJWT(username string, duration time.Duration) (string, error) {
	logrus.Infof("Gerando JWT para o usuário: %s com duração: %v", username, duration)
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Tokens structure for grouping Access and Refresh Tokens
type Tokens struct {
	AccessToken  string
	RefreshToken string
}
