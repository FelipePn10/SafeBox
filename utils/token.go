// Package utils provides utilities for handling Two-Factor Authentication (2FA)
package utils

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Algorithm representa os algoritmos suportados para 2FA
type Algorithm string

const (
	AlgorithmSHA1   Algorithm = "SHA1"
	AlgorithmSHA256 Algorithm = "SHA256"
	AlgorithmSHA512 Algorithm = "SHA512"
)

// Errors
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

// TwoFactorConfig define a configuração para gerar segredos 2FA
type TwoFactorConfig struct {
	Issuer         string     // Nome do emissor que aparecerá no app autenticador
	SecretSize     int        // Tamanho do segredo em bytes
	ValidityPeriod uint       // Período de validade do código em segundos
	Digits         otp.Digits // Número de dígitos no código
	Algorithm      Algorithm  // Algoritmo de hash utilizado
}

// DefaultTwoFactorConfig retorna a configuração padrão para 2FA
func DefaultTwoFactorConfig() TwoFactorConfig {
	return TwoFactorConfig{
		Issuer:         "SafeBox",
		SecretSize:     32,
		ValidityPeriod: 30,
		Digits:         otp.DigitsSix,
		Algorithm:      AlgorithmSHA1,
	}
}

// Generate2FASecret gera um segredo 2FA e uma URL para o QR Code
func Generate2FASecret(email string, config TwoFactorConfig) (secret, url string, err error) {
	if email == "" {
		return "", "", fmt.Errorf("email não pode estar vazio")
	}

	// Validar configuração
	if config.SecretSize < 16 {
		return "", "", fmt.Errorf("tamanho do segredo deve ser pelo menos 16 bytes")
	}

	// Gerar bytes aleatórios seguros
	bytes := make([]byte, config.SecretSize)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("erro ao gerar bytes aleatórios: %w", err)
	}

	secret = base32.StdEncoding.EncodeToString(bytes)

	// Converter algoritmo
	algorithm, err := config.Algorithm.toOTPAlgorithm()
	if err != nil {
		return "", "", err
	}

	// Gerar chave TOTP
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      config.Issuer,
		AccountName: email,
		Secret:      []byte(secret),
		Period:      config.ValidityPeriod,
		Digits:      config.Digits,
		Algorithm:   algorithm,
	})
	if err != nil {
		return "", "", fmt.Errorf("erro ao gerar chave TOTP: %w", err)
	}

	return secret, key.URL(), nil
}

// VerifyTwoFactorCode verifica se o código 2FA é válido
func VerifyTwoFactorCode(secret, code string) (bool, error) {
	return totp.ValidateCustom(
		code,
		secret,
		time.Now().UTC(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    6,
			Algorithm: otp.AlgorithmSHA1,
		},
	)
}

// ValidateWithTimeWindow verifica o código 2FA com uma janela de tempo personalizada
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
