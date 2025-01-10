package middlewares

import (
	"SafeBox/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ValidateTokenMiddleware valida o token OAuth
func ValidateTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token não fornecido"})
			c.Abort()
			return
		}

		_, err := utils.ValidateOAuthToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		c.Next()
	}
}
