// routes/routes.go
package routes

import (
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/services"
	"SafeBox/storage"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics estrutura para manter todas as métricas do Prometheus
type Metrics struct {
	uploads   prometheus.Counter
	downloads prometheus.Counter
	deletes   prometheus.Counter
}

// newMetrics inicializa e registra todas as métricas do Prometheus
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

	prometheus.MustRegister(
		m.uploads,
		m.downloads,
		m.deletes,
	)

	return m
}

// AppControllers agrupa todos os controladores da aplicação
type AppControllers struct {
	Auth      *controllers.AuthController
	Backup    *controllers.BackupController
	TwoFactor *controllers.TwoFactorController
	File      *controllers.FileController
}

// Config contém todas as configurações necessárias para as rotas
type Config struct {
	Echo        *echo.Echo
	Controllers AppControllers
	Metrics     *Metrics
}

// NewRouteConfig cria uma nova configuração de rotas com todas as dependências necessárias
func NewRouteConfig(authService *services.AuthService, backupService *services.BackupService,
	twoFactorService *services.TwoFactorService) (*echo.Echo, error) {
	e := echo.New()

	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("erro ao carregar arquivo .env: %w", err)
	}

	storage, err := setupStorage()
	if err != nil {
		return nil, fmt.Errorf("erro ao configurar storage: %w", err)
	}

	// Inicializa o middleware de autenticação
	authMiddleware := middlewares.NewAuthMiddleware(authService)

	controllers := AppControllers{
		Auth:      controllers.NewAuthController(authService),
		Backup:    controllers.NewBackupController(backupService),
		TwoFactor: controllers.NewTwoFactorController(),
		File:      controllers.NewFileController(storage),
	}

	metrics := newMetrics()

	config := &Config{
		Echo:        e,
		Controllers: controllers,
		Metrics:     metrics,
	}

	// Registra as rotas usando o novo middleware
	if err := registerRoutes(config, authMiddleware); err != nil {
		return nil, fmt.Errorf("erro ao registrar rotas: %w", err)
	}

	return e, nil
}

// setupStorage inicializa o backend de armazenamento
func setupStorage() (storage.Storage, error) {
	config := storage.R2Config{
		Bucket:          os.Getenv("R2_BUCKET_NAME"),
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuração R2 inválida: %w", err)
	}

	return storage.NewR2Storage(config)
}

// registerRoutes configura todas as rotas da aplicação
func registerRoutes(config *Config, auth *middlewares.AuthMiddleware) error {
	e := config.Echo
	c := config.Controllers
	m := config.Metrics

	public := e.Group("")
	registerPublicRoutes(public, c.Auth)

	// Usa o novo middleware de autenticação
	api := e.Group("/api", auth.RequireAuth())
	registerAPIRoutes(api, c)

	// Usa o novo middleware de verificação de plano
	files := api.Group("/files", middlewares.CheckUserPlan())
	registerFileRoutes(files, c.File, m)

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	return nil
}

// registerPublicRoutes registra todas as rotas públicas
func registerPublicRoutes(g *echo.Group, auth *controllers.AuthController) {
	g.POST("/register", auth.Register)
	g.POST("/login", auth.Login)
}

// registerAPIRoutes registra todas as rotas da API protegida
func registerAPIRoutes(g *echo.Group, c AppControllers) {
	g.POST("/backup", c.Backup.Backup)

	// Rotas 2FA
	twoFA := g.Group("/2fa")
	twoFA.POST("/setup", c.TwoFactor.Setup2FA)
	twoFA.POST("/enable", c.TwoFactor.Enable2FA)
}

// registerFileRoutes registra todas as rotas de arquivos
func registerFileRoutes(g *echo.Group, fc *controllers.FileController, m *Metrics) {
	g.POST("/upload", func(c echo.Context) error {
		m.uploads.Inc()
		return fc.Upload(c)
	})

	g.GET("/download/:id", func(c echo.Context) error {
		m.downloads.Inc()
		return fc.Download(c)
	})

	g.DELETE("/:id", func(c echo.Context) error {
		m.deletes.Inc()
		return fc.Delete(c)
	})
}
