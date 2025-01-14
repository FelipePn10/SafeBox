// middlewares/middleware.go
package middlewares

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// AuthConfig define as configurações para o middleware de autenticação
type AuthConfig struct {
	RequireToken      bool
	Require2FA        bool
	RequirePermission []models.Permission
}

// UserPlanConfig define as configurações para verificação do plano do usuário
type UserPlanConfig struct {
	AllowedPlans []string
}

// TokenValidator interface para validação de tokens
type TokenValidator interface {
	ValidateToken(token string) (*utils.TokenClaims, error)
}

// AuthMiddleware implementa as funções de autenticação
type AuthMiddleware struct {
	tokenValidator TokenValidator
	userRepo       repositories.UserRepository
}

// NewAuthMiddleware cria uma nova instância do middleware de autenticação
func NewAuthMiddleware(validator TokenValidator, userRepo repositories.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: validator,
		userRepo:       userRepo,
	}
}

// RequireAuth retorna um middleware que requer apenas autenticação
func (am *AuthMiddleware) RequireAuth() echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequireToken: true,
	})
}

// RequireAuthWith2FA retorna um middleware que requer autenticação e 2FA
func (am *AuthMiddleware) RequireAuthWith2FA() echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequireToken: true,
		Require2FA:   true,
	})
}

// RequirePermissions retorna um middleware que requer autenticação e permissões específicas
func (am *AuthMiddleware) RequirePermissions(permissions ...models.Permission) echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequireToken:      true,
		RequirePermission: permissions,
	})
}

// WithConfig retorna um middleware com configurações personalizadas
func (am *AuthMiddleware) WithConfig(config AuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.RequireToken {
				token := extractToken(c)
				if token == "" {
					return echo.NewHTTPError(http.StatusUnauthorized, "Token de autorização não encontrado")
				}

				claims, err := am.tokenValidator.ValidateToken(token)
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Token inválido: %v", err))
				}

				user, err := am.userRepo.FindByUsername(claims.Username)
				if err != nil {
					return echo.NewHTTPError(http.StatusUnauthorized, "Usuário não encontrado")
				}

				c.Set("user", user)

				if config.Require2FA {
					if err := validate2FA(c, user); err != nil {
						return err
					}
				}

				if len(config.RequirePermission) > 0 {
					if err := validatePermissions(user, config.RequirePermission); err != nil {
						return err
					}
				}
			}

			return next(c)
		}
	}
}

// CheckUserPlan verifica o plano e limites de armazenamento do usuário
func CheckUserPlan() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*models.User)
			if !ok {
				return echo.NewHTTPError(http.StatusInternalServerError, "Usuário não encontrado no contexto")
			}

			if user.Plan == "free" && user.StorageUsed >= user.StorageLimit {
				return echo.NewHTTPError(http.StatusForbidden, "Limite de armazenamento excedido")
			}

			return next(c)
		}
	}
}

// Funções auxiliares privadas
func extractToken(c echo.Context) string {
	token := c.Request().Header.Get("Authorization")
	return strings.TrimPrefix(token, "Bearer ")
}

func validate2FA(c echo.Context, user *models.User) error {
	code := c.Request().Header.Get("X-2FA-Code")
	if code == "" {
		return echo.NewHTTPError(http.StatusForbidden, "Código 2FA necessário")
	}

	isValid, err := utils.VerifyTwoFactorCode(user.TwoFASecret, code)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Erro na validação 2FA: %v", err))
	}
	if !isValid {
		return echo.NewHTTPError(http.StatusForbidden, "Código 2FA inválido")
	}

	return nil
}

func validatePermissions(user *models.User, requiredPermissions []models.Permission) error {
	for _, required := range requiredPermissions {
		if !hasPermission(user.Permissions, required) {
			return echo.NewHTTPError(http.StatusForbidden, "Permissão negada")
		}
	}
	return nil
}

func hasPermission(permissions []models.Permission, target models.Permission) bool {
	for _, permission := range permissions {
		if permission == target {
			return true
		}
	}
	return false
}
