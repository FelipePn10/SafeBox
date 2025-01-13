package routes

import (
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/storage"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func setupStorage() (storage.Storage, error) {
	bucketName := os.Getenv("R2_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("environment variable R2_BUCKET_NAME not set")
	}

	s3Storage, err := storage.NewR2Storage(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to configure S3 storage: %w", err)
	}

	return s3Storage, nil
}

func RegisterAPIRoutes(e *echo.Echo) {
	e.POST("/login", controllers.LoginController)

	protected := e.Group("/api")
	protected.Use(middlewares.AuthMiddleware())

	protected.GET("/protected-resource", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{"message": "Access granted"})
	})
}

// RegisterBackupRoutes configura rotas relacionadas a backups
func RegisterBackupRoutes(e *echo.Echo) {
	s3Storage, err := setupStorage()
	if err != nil {
		log.Fatalf("Critical error: %v", err)
	}

	backupController := controllers.NewBackupController(s3Storage)

	backup := e.Group("/backup", middlewares.AuthMiddleware())
	backup.POST("/gallery", backupController.BackupGallery)
	backup.POST("/whatsapp", backupController.BackupWhatsApp)
	backup.POST("/app", backupController.BackupApp)
}
