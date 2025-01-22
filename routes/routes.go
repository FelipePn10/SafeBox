package routes

import (
	"SafeBox/controllers"
	"SafeBox/handlers"
	"SafeBox/middlewares"
	"SafeBox/repositories"
	"SafeBox/services"
	"SafeBox/storage"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Metrics struct {
	uploads   prometheus.Counter
	downloads prometheus.Counter
	deletes   prometheus.Counter
}

func newMetrics() *Metrics {
	m := &Metrics{
		uploads: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "safebox",
			Name:      "uploads_total",
			Help:      "Total number of uploads",
		}),
		downloads: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "safebox",
			Name:      "downloads_total",
			Help:      "Total number of downloads",
		}),
		deletes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "safebox",
			Name:      "deletes_total",
			Help:      "Total number of deletions",
		}),
	}

	prometheus.MustRegister(m.uploads, m.downloads, m.deletes)
	return m
}

type AppControllers struct {
	Backup *controllers.BackupController
	File   *controllers.FileController
	OAuth  *controllers.OAuthController
}

type Config struct {
	Echo        *echo.Echo
	Controllers AppControllers
	Metrics     *Metrics
}

func NewRouteConfig(authService *services.AuthService, backupService *services.BackupService, db *gorm.DB) (*echo.Echo, error) {
	logger := logrus.WithField("component", "RouteConfig")

	if err := godotenv.Load(); err != nil {
		logger.WithError(err).Error("Failed to load .env file")
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	storage, err := setupStorage()
	if err != nil {
		logger.WithError(err).Error("Failed to setup storage")
		return nil, fmt.Errorf("failed to setup storage: %w", err)
	}

	// Initialize UserRepository
	userRepo := repositories.NewUserRepository(db)

	// Initialize OAuth controller with the interface
	oauthController, err := controllers.NewOAuthController(userRepo)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize OAuth controller")
		return nil, fmt.Errorf("failed to initialize OAuth controller: %w", err)
	}

	e := echo.New()

	authMiddleware := middlewares.NewAuthMiddleware(userRepo)

	controllers := AppControllers{
		Backup: controllers.NewBackupController(storage, backupService.GetBackupRepo()),
		File:   controllers.NewFileController(storage),
		OAuth:  oauthController,
	}

	metrics := newMetrics()

	config := &Config{
		Echo:        e,
		Controllers: controllers,
		Metrics:     metrics,
	}

	if err := registerRoutes(config, authMiddleware); err != nil {
		logger.WithError(err).Error("Failed to register routes")
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	return e, nil
}

func setupStorage() (storage.Storage, error) {
	config := storage.R2Config{
		Bucket:          os.Getenv("R2_BUCKET_NAME"),
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid R2 configuration: %w", err)
	}

	return storage.NewR2Storage(config)
}

func registerRoutes(config *Config, auth *middlewares.AuthMiddleware) error {
	e := config.Echo

	// Public routes
	public := e.Group("")
	{
		public.GET("/oauth/login", oauthHandler.Login)
		public.GET("/oauth/callback", oauthHandler.Callback)
		public.GET("/health", handleHealth)
	}

	// Protected API routes
	api := e.Group("/api", auth.RequireAuth())
	{
		// Backup routes
		api.POST("/backup", config.Controllers.Backup.Backup)

		// File routes
		api.POST("/upload", config.Controllers.File.Upload)
		api.GET("/download/:id", config.Controllers.File.Download)
		api.DELETE("/delete/:id", config.Controllers.File.Delete)
	}

	// Metrics route
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	return nil
}

// handleHealth handles the health check endpoint
func handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"info":   "Service is healthy",
	})
}
