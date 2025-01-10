package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logrus.Error("Error request: ", err)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process request"})
		}
	}
}
