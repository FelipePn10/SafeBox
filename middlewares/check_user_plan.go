package middlewares

import (
	"SafeBox/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckUserPlanMiddleware checks the user's plan and storage limits
func CheckUserPlanMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		// Check if the user is on the free plan and if the storage limit has been exceeded
		if user.Plan == "free" && user.StorageUsed >= user.StorageLimit {
			c.JSON(http.StatusForbidden, gin.H{"error": "Storage limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
