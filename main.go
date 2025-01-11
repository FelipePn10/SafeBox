package main

import (
	"SafeBox/config"
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/repositories"
	"SafeBox/services"
	"SafeBox/storage"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	uploadsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "safebox_uploads_total",
		Help: "Total de uploads realizados",
	})
	downloadsCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "safebox_downloads_total",
		Help: "Total de downloads realizados",
	})
	deletesCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "safebox_deletes_total",
		Help: "Total de exclusões realizadas",
	})
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	prometheus.MustRegister(uploadsCounter, downloadsCounter, deletesCounter)
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Fatal("Error loading environment variables: ", err)
	}

	// Load configuration
	cfg := config.LoadConfig()
	if cfg == nil {
		logrus.Fatal("Failed to load configurations")
	}
	logrus.Infof("Configurations successfully loaded: %+v", cfg)

	// Initialize database connection
	repositories.InitDB()
	dbConn := repositories.DBConection
	userRepo := repositories.NewUserRepository(dbConn)

	// Initialize services
	authService := services.NewAuthService(userRepo)

	// Initialize file storage
	var fileStorage storage.Storage
	switch os.Getenv("STORAGE_TYPE") {
	case "r2":
		r2Storage, err := storage.NewR2Storage(os.Getenv("R2_BUCKET_NAME"))
		if err != nil {
			logrus.Fatal("Erro ao configurar Cloudflare R2: ", err)
		}
		fileStorage = r2Storage
	default:
		fileStorage = storage.NewLocalStorage("./uploads")
	}

	fileController := controllers.NewFileController(fileStorage)

	// Set up Gin router
	r := gin.Default()

	// Apply global middlewares
	r.Use(middlewares.RecoveryMiddleware())
	r.Use(middlewares.ValidateTokenMiddleware())
	r.Use(middlewares.ErrorHandler())

	// Public routes
	authController := controllers.NewAuthController(authService)
	r.POST("/register", authController.Register)
	r.POST("/login", authController.Login)

	// Protected routes
	authGroup := r.Group("/")
	//authGroup.Use(middlewares.AuthMiddleware())
	authGroup.Use(middlewares.CheckUserPlanMiddleware()) // Adicionar middleware para verificar o plano do usuário e limites de armazenamento
	{
		authGroup.POST("/upload", fileController.Upload)
		authGroup.GET("/download/:id", fileController.Download)
		authGroup.DELETE("/files/:id", fileController.Delete)
	}
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start server
	if err := r.Run(":8080"); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}
