package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pquerna/otp"
)

type Algorithm string

const (
	AlgorithmSHA1   Algorithm = "SHA1"
	AlgorithmSHA256 Algorithm = "SHA256"
	AlgorithmSHA512 Algorithm = "SHA512"
)

var (
	ErrInvalidAlgorithm = fmt.Errorf("algoritmo inválido")
	ErrInvalidSecret    = fmt.Errorf("segredo inválido")
	ErrInvalidCode      = fmt.Errorf("código inválido")
	ErrGeneratingKey    = fmt.Errorf("erro ao gerar chave")
)

func (a Algorithm) toOTPAlgorithm() (otp.Algorithm, error) {
	switch a {
	case AlgorithmSHA1:
		return otp.AlgorithmSHA1, nil
	case AlgorithmSHA256:
		return otp.AlgorithmSHA256, nil
	case AlgorithmSHA512:
		return otp.AlgorithmSHA512, nil
	default:
		return otp.AlgorithmSHA1, ErrInvalidAlgorithm
	}
}

type TokenClaims struct {
	Username       string    `json:"username"`
	ExpirationTime time.Time `json:"exp"`
	Issuer         string    `json:"iss"`
	Subject        string    `json:"sub"`
	IssuedAt       time.Time `json:"iat"`
	ID             string    `json:"jti"`
}

func (c *TokenClaims) Valid() error {
	if time.Now().After(c.ExpirationTime) {
		return errors.New("token expirado")
	}
	return nil
}

func ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
