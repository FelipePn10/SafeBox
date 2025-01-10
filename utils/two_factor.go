package utils

import (
	"github.com/pquerna/otp/totp"
)

// VerifyTwoFactorCode checks the 2FA code
func VerifyTwoFactorCode(secret, code string) bool {
	return totp.Validate(code, secret)
}
