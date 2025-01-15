package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
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
type TwoFactorConfig struct {
	Issuer         string
	SecretSize     int
	ValidityPeriod uint
	Digits         otp.Digits
	Algorithm      Algorithm
}

func (c *TokenClaims) Valid() error {
	if time.Now().After(c.ExpirationTime) {
		return errors.New("token expirado")
	}
	return nil
}

func DefaultTwoFactorConfig() TwoFactorConfig {
	return TwoFactorConfig{
		Issuer:         "SafeBox",
		SecretSize:     32,
		ValidityPeriod: 30,
		Digits:         otp.DigitsSix,
		Algorithm:      AlgorithmSHA1,
	}
}

func ValidateWithTimeWindow(code, secret string, window int, config TwoFactorConfig) (bool, error) {
	if code == "" || secret == "" {
		return false, ErrInvalidCode
	}

	if window < 0 {
		return false, fmt.Errorf("janela de tempo deve ser positiva")
	}

	algorithm, err := config.Algorithm.toOTPAlgorithm()
	if err != nil {
		return false, err
	}

	opts := totp.ValidateOpts{
		Period:    config.ValidityPeriod,
		Skew:      uint(window),
		Digits:    config.Digits,
		Algorithm: algorithm,
	}

	return totp.ValidateCustom(code, secret, time.Now(), opts)
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
