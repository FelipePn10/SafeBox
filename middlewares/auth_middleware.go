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

type AuthConfig struct {
	RequireToken      bool
	RequirePermission []string // Usamos strings para as permissões
}

type UserPlanConfig struct {
	AllowedPlans []string
}

type TokenValidator interface {
	ValidateToken(token string) (*utils.TokenClaims, error)
}

type AuthMiddleware struct {
	tokenValidator TokenValidator
	userRepo       repositories.UserRepository
}

func NewAuthMiddleware(validator TokenValidator, userRepo repositories.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: validator,
		userRepo:       userRepo,
	}
}

func (am *AuthMiddleware) RequireAuth() echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequireToken: true,
	})
}

func (am *AuthMiddleware) RequirePermissions(permissions ...string) echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequireToken:      true,
		RequirePermission: permissions,
	})
}

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

func CheckUserPlan() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("user").(*models.OAuthUser)
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

func extractToken(c echo.Context) string {
	token := c.Request().Header.Get("Authorization")
	return strings.TrimPrefix(token, "Bearer ")
}

// Função para validar permissões
func validatePermissions(user *models.OAuthUser, requiredPermissions []string) error {
	// Converte as permissões do usuário para um slice de strings
	userPermissions := make([]string, len(user.Permissions))
	for i, p := range user.Permissions {
		userPermissions[i] = p.Name
	}

	// Verifica se o usuário tem todas as permissões necessárias
	for _, required := range requiredPermissions {
		if !hasPermission(userPermissions, required) {
			return echo.NewHTTPError(http.StatusForbidden, "Permissão negada: "+required)
		}
	}
	return nil
}

// Função auxiliar para verificar se uma permissão está presente
func hasPermission(permissions []string, target string) bool {
	for _, permission := range permissions {
		if permission == target {
			return true
		}
	}
	return false
}
