package utils

import (
	"github.com/pquerna/otp/totp"
)

// VerifyTwoFactorCode verifica o código 2FA
func VerifyTwoFactorCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
