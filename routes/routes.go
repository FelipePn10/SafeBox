package routes

import (
	"SafeBox/controllers"
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
			Help:      "Total de uploads realizados",
		}),
		downloads: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "safebox",
			Name:      "downloads_total",
			Help:      "Total de downloads realizados",
		}),
		deletes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "safebox",
			Name:      "deletes_total",
			Help:      "Total de exclusões realizadas",
		}),
	}

	prometheus.MustRegister(m.uploads, m.downloads, m.deletes)
	return m
}

type AppControllers struct {
	Backup *controllers.BackupController
	File   *controllers.FileController
}

type Config struct {
	Echo        *echo.Echo
	Controllers AppControllers
	Metrics     *Metrics
}

func NewRouteConfig(authService *services.AuthService, backupService *services.BackupService, userRepo repositories.UserRepository) (*echo.Echo, error) {
	e := echo.New()

	if err := godotenv.Load(); err != nil {
		logrus.WithError(err).Error("Erro ao carregar arquivo .env")
		return nil, fmt.Errorf("erro ao carregar arquivo .env: %w", err)
	}

	storage, err := setupStorage()
	if err != nil {
		logrus.WithError(err).Error("Erro ao configurar storage")
		return nil, fmt.Errorf("erro ao configurar storage: %w", err)
	}

	authMiddleware := middlewares.NewAuthMiddleware(authService, userRepo)

	controllers := AppControllers{
		Backup: controllers.NewBackupController(storage, backupService.GetBackupRepo()),
		File:   controllers.NewFileController(storage),
	}

	metrics := newMetrics()

	config := &Config{
		Echo:        e,
		Controllers: controllers,
		Metrics:     metrics,
	}

	if err := registerRoutes(config, authMiddleware); err != nil {
		logrus.WithError(err).Error("Erro ao registrar rotas")
		return nil, fmt.Errorf("erro ao registrar rotas: %w", err)
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
		logrus.WithError(err).Error("Configuração R2 inválida")
		return nil, fmt.Errorf("configuração R2 inválida: %w", err)
	}

	return storage.NewR2Storage(config)
}

func registerRoutes(config *Config, auth *middlewares.AuthMiddleware) error {
	e := config.Echo

	// Rotas públicas
	public := e.Group("")
	public.GET("/oauth/login", controllers.OAuthLogin)       // Inicia o fluxo OAuth
	public.GET("/oauth/callback", controllers.OAuthCallback) // Callback do OAuth
	public.GET("/health", func(c echo.Context) error {       // Health check
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Rotas protegidas
	api := e.Group("/api", auth.RequireAuth())
	api.POST("/backup", config.Controllers.Backup.Backup)
	api.POST("/upload", config.Controllers.File.Upload)
	api.GET("/download/:id", config.Controllers.File.Download)
	api.DELETE("/delete/:id", config.Controllers.File.Delete)

	// Métricas
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	return nil
}
