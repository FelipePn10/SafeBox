package routes

import (
	"SafeBox/handlers"
	"SafeBox/middlewares"
	"SafeBox/repositories"
	"SafeBox/services"

	// "SafeBox/storage"
	// "fmt"
	// "net/http"
	// "os"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Metrics struct {
	uploads   prometheus.Counter
	downloads prometheus.Counter
	deletes   prometheus.Counter
}

func newMetrics() *Metrics {
	return &Metrics{
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
}

type AppHandlers struct {
	OAuth  *handlers.OAuthHandler
	Backup *handlers.BackupHandler
	File   *handlers.FileHandler
}

func NewRouteConfig(
	authService *services.AuthService,
	backupService *services.BackupService,
	db *gorm.DB,
	oauthConfig *oauth2.Config,
) (*echo.Echo, error) {
	e := echo.New()

	// Initialize storage
	// //storage, err := setupStorage()
	// if err != nil {
	// 	return nil, fmt.Errorf("storage setup failed: %w", err)
	// }

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	//backupRepo := repositories.NewBackupRepository(db)

	// Initialize handlers
	oauthHandler := handlers.NewOAuthHandler(userRepo, oauthConfig)
	// backupHandler := handlers.NewBackupHandler(storage, backupRepo)
	// fileHandler := handlers.NewFileHandler(storage)

	// Register metrics
	metrics := newMetrics()
	prometheus.MustRegister(metrics.uploads, metrics.downloads, metrics.deletes)

	// Setup routes
	e.Use(middlewares.ErrorHandler())
	e.Use(middlewares.RecoveryMiddleware())

	// Public routes
	//e.GET("/health", handleHealth)
	e.GET("/oauth/login", oauthHandler.Login)
	e.GET("/oauth/callback", oauthHandler.Callback)

	// Protected routes
	api := e.Group("/api")
	api.Use(middlewares.NewAuthMiddleware(userRepo, oauthConfig).RequireAuth())
	{
		// api.POST("/backup", backupHandler.Backup)
		// api.POST("/upload", fileHandler.Upload)
		//api.GET("/download/:id", fileHandler.Download)
		//api.DELETE("/delete/:id", fileHandler.Delete)
	}

	// Metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	return e, nil
}

// func setupStorage() (storage.Storage, error) {
// 	cfg := storage.R2Config{
// 		Bucket:          os.Getenv("R2_BUCKET_NAME"),
// 		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
// 		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
// 		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
// 	}

// 	if err := cfg.Validate(); err != nil {
// 		return nil, fmt.Errorf("invalid R2 config: %w", err)
// 	}

// 	return storage.NewR2Storage(cfg)
// }

// func handleHealth(c echo.Context) error {
// 	return c.JSON(http.StatusOK, map[string]interface{}{
// 		"status":  "healthy",
// 		"version": "1.0.0",
// 	})
// }
