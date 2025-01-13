package main

import (
	"SafeBox/config"
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/repositories"
	"SafeBox/routes"
	"SafeBox/services"
	"SafeBox/storage"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	promRegister()
}

func promRegister() {
	prometheus.MustRegister(routes.UploadsCounter, routes.DownloadsCounter, routes.DeletesCounter)
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
	_ = authService

	// Initialize file storage using R2
	var fileStorage storage.Storage
	r2Storage, err := storage.NewR2Storage(os.Getenv("R2_BUCKET_NAME"))
	if err != nil {
		logrus.Fatal("Error configuring Cloudflare R2: ", err)
	}
	fileStorage = r2Storage

	fileController := controllers.NewFileController(fileStorage)
	_ = fileController

	// Set up Echo router
	e := echo.New()

	// Apply global middlewares
	e.Use(middlewares.RecoveryMiddleware())
	e.Use(middlewares.ValidateTokenMiddleware())
	e.Use(middlewares.ErrorHandler())

	// Register all routes
	routes.RegisterRoutes(e)

	// Start server
	if err := e.Start(":8080"); err != nil {
		logrus.Fatal("Error starting server: ", err)
	}
}
