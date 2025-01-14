// controllers/two_factor_controller.go

package controllers

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
	"net/http"

	"github.com/labstack/echo/v4"
)

type TwoFactorController struct{}

func NewTwoFactorController() *TwoFactorController {
	return &TwoFactorController{}
}

// Setup2FA gera e retorna o segredo 2FA e URL do QR code
func (c *TwoFactorController) Setup2FA(ctx echo.Context) error {
	userEmail := ctx.Get("user_email").(string)
	if userEmail == "" {
		return ctx.JSON(http.StatusUnauthorized, echo.Map{
			"error": "não autorizado",
		})
	}

	config := utils.DefaultTwoFactorConfig()
	secret, qrCodeURL, err := utils.Generate2FASecret(userEmail, config)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "falha ao gerar segredo 2FA",
		})
	}

	tempSecret := models.TempTwoFASecret{
		UserEmail: userEmail,
		Secret:    secret,
	}

	if err := repositories.DBConnection.Create(&tempSecret).Error; err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "falha ao armazenar segredo temporário",
		})
	}

	return ctx.JSON(http.StatusOK, echo.Map{
		"qr_code_url": qrCodeURL,
		"secret":      secret,
		"message":     "Escaneie o código QR com seu aplicativo autenticador",
	})
}

// Enable2FA valida o código inicial 2FA e habilita 2FA para o usuário
// Update the Enable2FA method to properly handle multiple return values
func (c *TwoFactorController) Enable2FA(ctx echo.Context) error {
	userEmail := ctx.Get("user_email").(string)
	code := ctx.FormValue("code")

	if code == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "código 2FA necessário",
		})
	}

	var tempSecret models.TempTwoFASecret
	if err := repositories.DBConnection.Where("user_email = ?", userEmail).First(&tempSecret).Error; err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "nenhuma configuração 2FA pendente encontrada",
		})
	}

	// Fix: Properly handle both return values
	isValid, err := utils.VerifyTwoFactorCode(tempSecret.Secret, code)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "erro ao verificar código 2FA",
		})
	}
	if !isValid {
		return ctx.JSON(http.StatusBadRequest, echo.Map{
			"error": "código 2FA inválido",
		})
	}

	if err := repositories.DBConnection.Model(&models.OAuthUser{}).
		Where("email = ?", userEmail).
		Update("two_fa_secret", tempSecret.Secret).Error; err != nil {
		return ctx.JSON(http.StatusInternalServerError, echo.Map{
			"error": "falha ao habilitar 2FA",
		})
	}

	repositories.DBConnection.Delete(&tempSecret)

	return ctx.JSON(http.StatusOK, echo.Map{
		"message": "2FA habilitado com sucesso",
	})
}
