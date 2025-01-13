package routes

import (
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/services"
	"SafeBox/storage"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	UploadsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "safebox_uploads_total",
		Help: "Total de uploads realizados",
	})
	DownloadsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "safebox_downloads_total",
		Help: "Total de downloads realizados",
	})
	DeletesCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "safebox_deletes_total",
		Help: "Total de exclusões realizadas",
	})
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

// RegisterRoutes configures all routes for the application
func RegisterAllRoutes(e *echo.Echo, authService *services.AuthService, fileController *controllers.FileController) {
	// Register API routes
	RegisterAPIRoutes(e)

	// Register Backup routes
	RegisterBackupRoutes(e)

	// Public routes
	authController := controllers.NewAuthController(authService)
	e.POST("/register", authController.Register)
	e.POST("/login", authController.Login)

	// Protected routes
	authGroup := e.Group("/")
	authGroup.Use(middlewares.CheckUserPlanMiddleware())
	{
		authGroup.POST("/upload", func(c echo.Context) error {
			UploadsCounter.Inc()
			return fileController.Upload(c)
		})
		authGroup.GET("/download/:id", func(c echo.Context) error {
			DownloadsCounter.Inc()
			return fileController.Download(c)
		})
		authGroup.DELETE("/files/:id", func(c echo.Context) error {
			DeletesCounter.Inc()
			return fileController.Delete(c)
		})
	}
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}

func RegisterAPIRoutes(e *echo.Echo) {
	e.POST("/login", controllers.LoginController)

	protected := e.Group("/api")
	protected.Use(middlewares.AuthMiddleware())

	protected.GET("/protected-resource", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{"message": "Access granted"})
	})
}

// RegisterBackupRoutes configures backup related routes
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
