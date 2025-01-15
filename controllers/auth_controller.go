package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	jwtSecret = []byte(os.Getenv("JWT_SECRET")) // Chave secreta para assinar o JWT
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),     // Client ID do Google
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"), // Client Secret do Google
		RedirectURL:  os.Getenv("OAUTH_REDIRECT_URL"),   // URL de redirecionamento
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	stateTokens = make(map[string]bool) // Para armazenar e validar tokens de estado
)

// OAuthLogin redireciona o usuário para a página de autenticação do Google
func OAuthLogin(c echo.Context) error {
	state := generateStateToken()
	url := oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// OAuthCallback lida com a resposta do Google após a autenticação
func OAuthCallback(c echo.Context) error {
	// Valida o token de estado
	state := c.QueryParam("state")
	if !stateTokens[state] {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid state token",
		})
	}
	delete(stateTokens, state) // Remove o token de estado após o uso

	// Obtém o código de autorização
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "authorization code not provided",
		})
	}

	// Troca o código por um token de acesso
	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "failed to exchange authorization code",
		})
	}

	// Obtém as informações do usuário
	userInfo, err := getUserInfo(token)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "failed to retrieve user information",
		})
	}

	// Gera um JWT para o usuário
	jwtToken, err := generateJWT(userInfo["email"].(string))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "failed to generate JWT",
		})
	}

	// Retorna o JWT para o cliente
	return c.JSON(http.StatusOK, echo.Map{
		"token": jwtToken,
		"user":  userInfo,
	})
}

// generateStateToken gera um token de estado seguro
func generateStateToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	token := base64.StdEncoding.EncodeToString(b)
	stateTokens[token] = true
	return token
}

// getUserInfo obtém as informações do usuário a partir do token de acesso
func getUserInfo(token *oauth2.Token) (map[string]interface{}, error) {
	client := oauthConf.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch user info: status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

// generateJWT cria um JWT para o email do usuário
func generateJWT(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}
