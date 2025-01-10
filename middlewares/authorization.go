package middlewares

import (
	"SafeBox/models"

	"github.com/gin-gonic/gin"
)

func Authorize(permissions ...models.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		oauthUser, exists := c.Get("oauth_user")
		if !exists {
			c.JSON(401, gin.H{"error": "Usuário não autenticado"})
			c.Abort()
			return
		}

		user := oauthUser.(models.OAuthUser)
		for _, permission := range permissions {
			if !hasPermission(user.Permissions, permission) {
				c.JSON(403, gin.H{"error": "Permissão negada"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func hasPermission(permissions []models.Permission, target models.Permission) bool {
	for _, permission := range permissions {
		if permission == target {
			return true
		}
	}
	return false
}
