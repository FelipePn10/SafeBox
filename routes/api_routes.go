package routes

import (
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"net/http"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes initializes all API routes.
func RegisterRoutes(e *echo.Echo) {
	// Public routes
	e.POST("/login", controllers.LoginController)

	// Protected routes
	protected := e.Group("/api")
	protected.Use(middlewares.AuthMiddleware())
	protected.GET("/protected-resource", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{
			"message": "Access granted to protected resource",
		})
	})
}
