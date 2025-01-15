package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var jwtSecret []byte

// init carrega a chave secreta do arquivo .env
func init() {
	// Carrega as variáveis de ambiente do arquivo .env
	if err := godotenv.Load(); err != nil {
		logrus.Fatal("Erro ao carregar o arquivo .env")
	}

	// Obtém a chave secreta do ambiente
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		logrus.Fatal("JWT_SECRET não encontrado no arquivo .env")
	}

	jwtSecret = []byte(secret)
}

// Claims define os dados do token JWT
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Tokens structure for grouping Access and Refresh Tokens
type Tokens struct {
	AccessToken  string `json:"access_token"`  // Token de acesso
	RefreshToken string `json:"refresh_token"` // Token de refresh
}

// GenerateOAuthToken generates an access token and a refresh token
func GenerateOAuthToken(ctx context.Context, username string) (*Tokens, error) {
	accessToken, err := generateJWT(ctx, username, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar Access Token: %w", err)
	}

	refreshToken, err := generateJWT(ctx, username, 7*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar Refresh Token: %w", err)
	}

	return &Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateOAuthToken validates a JWT token and returns user data
func ValidateOAuthToken(ctx context.Context, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid || claims.Username == "" {
		return nil, errors.New("token inválido ou usuário não encontrado")
	}

	return claims, nil
}

func generateJWT(ctx context.Context, username string, duration time.Duration) (string, error) {
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
