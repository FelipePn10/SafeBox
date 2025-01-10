package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

var jwtSecret = []byte("your_secret_key")

// LoginController handles user login and returns a JWT.
func LoginController(c echo.Context) error {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "invalid request payload",
		})
	}

	if !authenticateUser(req.Email, req.Password) {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error": "invalid email or password",
		})
	}

	token, err := generateJWT(req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "failed to generate token",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
	})
}

// authenticateUser is a mock function to validate email and password.
func authenticateUser(email, password string) bool {
	// Replace with real authentication logic.
	return email == "test@example.com" && password == "password123"
}

// generateJWT creates a JWT for the given email.
func generateJWT(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   jwt.TimeFunc().Add(24 * time.Hour).Unix(),
	})

	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}
