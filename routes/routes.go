package routes

import (
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/services"
	"SafeBox/storage"
	"fmt"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics definitions
var (
	metrics = struct {
		uploads   prometheus.Counter
		downloads prometheus.Counter
		deletes   prometheus.Counter
	}{
		uploads: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "safebox_uploads_total",
			Help: "Total de uploads realizados",
		}),
		downloads: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "safebox_downloads_total",
			Help: "Total de downloads realizados",
		}),
		deletes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "safebox_deletes_total",
			Help: "Total de exclusões realizadas",
		}),
	}
)

// RouteConfig holds all dependencies needed for route configuration
type RouteConfig struct {
	Echo           *echo.Echo
	AuthService    *services.AuthService
	FileController *controllers.FileController
	Storage        storage.Storage
}

// NewRouteConfig creates a new RouteConfig with all necessary dependencies
func NewRouteConfig(e *echo.Echo, authService *services.AuthService, fileController *controllers.FileController) (*RouteConfig, error) {
	storage, err := setupStorage()
	if err != nil {
		return nil, fmt.Errorf("failed to setup storage: %w", err)
	}

	return &RouteConfig{
		Echo:           e,
		AuthService:    authService,
		FileController: fileController,
		Storage:        storage,
	}, nil
}

// setupStorage initializes the storage backend
func setupStorage() (storage.Storage, error) {
	bucketName := os.Getenv("R2_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("environment variable R2_BUCKET_NAME not set")
	}

	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")

	if accountID == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("one or more R2 configuration environment variables are not set")
	}

	r2Config := storage.R2Config{
		Bucket:          bucketName,
		AccountID:       accountID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}

	s3Storage, err := storage.NewR2Storage(r2Config)
	if err != nil {
		return nil, fmt.Errorf("failed to configure R2 storage: %w", err)
	}

	return s3Storage, nil
}

// RegisterAllRoutes configures all routes for the application
func (rc *RouteConfig) RegisterAllRoutes() {
	// Register prometheus metrics
	prometheus.MustRegister(metrics.uploads, metrics.downloads, metrics.deletes)

	// Public routes
	rc.registerAuthRoutes()

	// Protected routes
	rc.registerFileRoutes()
	rc.registerBackupRoutes()
	rc.registerMetricsRoute()
}

// registerAuthRoutes configures authentication-related routes
func (rc *RouteConfig) registerAuthRoutes() {
	authController := controllers.NewAuthController(rc.AuthService)

	rc.Echo.POST("/register", authController.Register)
	rc.Echo.POST("/login", authController.Login)
}

// registerFileRoutes configures file management routes
func (rc *RouteConfig) registerFileRoutes() {
	files := rc.Echo.Group("/files")
	files.Use(middlewares.RequireAuth())
	files.Use(middlewares.CheckUserPlanMiddleware())

	files.POST("/upload", func(c echo.Context) error {
		metrics.uploads.Inc()
		return rc.FileController.Upload(c)
	})

	files.GET("/download/:id", func(c echo.Context) error {
		metrics.downloads.Inc()
		return rc.FileController.Download(c)
	})

	files.DELETE("/:id", func(c echo.Context) error {
		metrics.deletes.Inc()
		return rc.FileController.Delete(c)
	})
}

// registerBackupRoutes configures backup-related routes
func (rc *RouteConfig) registerBackupRoutes() {
	backupController := controllers.NewBackupController(rc.Storage)

	backup := rc.Echo.Group("/backup")
	backup.Use(middlewares.RequireAuth())

	backup.POST("/gallery", backupController.BackupGallery)
	backup.POST("/whatsapp", backupController.BackupWhatsApp)
	backup.POST("/app", backupController.BackupApp)
}

// registerMetricsRoute configures the Prometheus metrics endpoint
func (rc *RouteConfig) registerMetricsRoute() {
	rc.Echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
}
