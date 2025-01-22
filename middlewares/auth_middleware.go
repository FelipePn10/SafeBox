package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"SafeBox/models"
	"SafeBox/repositories"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

type AuthConfig struct {
	RequireAuth    bool
	RequiredScopes []string
	RequiredPlan   string
	CheckStorage   bool
}

type AuthMiddleware struct {
	oauthConfig *oauth2.Config
	userRepo    repositories.UserRepository
}

func NewAuthMiddleware(userRepo repositories.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		userRepo: userRepo,
	}
}

func (am *AuthMiddleware) RequireAuth(scopes ...string) echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequireAuth:    true,
		RequiredScopes: scopes,
	})
}

func (am *AuthMiddleware) RequirePlan(plan string) echo.MiddlewareFunc {
	return am.WithConfig(AuthConfig{
		RequiredPlan: plan,
	})
}

func (am *AuthMiddleware) WithConfig(config AuthConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.RequireAuth {
				user, err := am.validateAuth(c)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, errorResponse(err))
				}
				c.Set("user", user)
			}

			if config.RequiredPlan != "" {
				if err := am.validatePlan(c); err != nil {
					return c.JSON(http.StatusForbidden, errorResponse(err))
				}
			}

			if config.CheckStorage {
				if err := am.validateStorage(c); err != nil {
					return c.JSON(http.StatusForbidden, errorResponse(err))
				}
			}

			return next(c)
		}
	}
}

func (am *AuthMiddleware) validateAuth(c echo.Context) (*models.OAuthUser, error) {
	token := extractAccessToken(c)
	if token == "" {
		return nil, errors.New("authorization token required")
	}

	// Verifica token com Google
	userInfo, err := am.verifyGoogleToken(c.Request().Context(), token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %v", err)
	}

	// Busca usuário no banco
	user, err := am.userRepo.FindByEmail(userInfo["email"].(string))
	if err != nil {
		return nil, errors.New("user not registered")
	}

	return user, nil
}

func (am *AuthMiddleware) verifyGoogleToken(ctx context.Context, token string) (map[string]interface{}, error) {
	client := am.oauthConfig.Client(ctx, &oauth2.Token{AccessToken: token})
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status: %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return userInfo, nil
}

func (am *AuthMiddleware) validatePlan(c echo.Context) error {
	user := getAuthUser(c)
	if user.Plan != c.Get("required_plan").(string) {
		return fmt.Errorf("plan '%s' required", c.Get("required_plan"))
	}
	return nil
}

func (am *AuthMiddleware) validateStorage(c echo.Context) error {
	user := getAuthUser(c)
	if user.StorageUsed >= user.StorageLimit {
		return errors.New("storage limit exceeded")
	}
	return nil
}

func extractAccessToken(c echo.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if parts := strings.Split(authHeader, " "); len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func getAuthUser(c echo.Context) *models.OAuthUser {
	return c.Get("user").(*models.OAuthUser)
}

func errorResponse(err error) map[string]interface{} {
	return map[string]interface{}{
		"error":     err.Error(),
		"success":   false,
		"timestamp": time.Now().UTC(),
	}
}
