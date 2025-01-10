package middlewares

import (
	"SafeBox/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckUserPlanMiddleware verifica o plano do usuário e limites de armazenamento
func CheckUserPlanMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		// Verificar se o usuário está no plano free e se o limite de armazenamento foi excedido
		if user.Plan == "free" && user.StorageUsed >= user.StorageLimit {
			c.JSON(http.StatusForbidden, gin.H{"error": "Storage limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
