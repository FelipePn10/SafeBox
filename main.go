package main

import (
	"SafeBox/config"
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/repositories"
	"SafeBox/routes"
	"SafeBox/services"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func init() {
	// Configure logging
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
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

	repositories.InitDB(repositories.DBConfig{
		DB_HOST:     cfg.DBHost,
		DB_USER:     cfg.DBUser,
		DB_PASSWORD: cfg.DBPassword,
		DB_NAME:     cfg.DBName,
		DB_PORT:     fmt.Sprintf("%d", cfg.DBPort),
	})
	// Initialize database connection
	dbConn := repositories.DBConection
	userRepo := repositories.NewUserRepository(dbConn)

	// Initialize services
	authService := services.NewAuthService(userRepo)

	// Initialize controllers
	fileController := controllers.NewFileController(nil) // Storage will be initialized by RouteConfig

	// Set up Echo router
	e := echo.New()

	// Apply global middlewares
	e.Use(middlewares.RecoveryMiddleware())
	e.Use(middlewares.ErrorHandler())

	// Create route configuration
	routeConfig, err := routes.NewRouteConfig(e, authService, fileController)
	if err != nil {
		logrus.Fatalf("Failed to create route config: %v", err)
	}

	// Register all routes
	routeConfig.RegisterAllRoutes()

	// Configure custom error handler
	//e.HTTPErrorHandler = middlewares.CustomErrorHandler

	// Start server
	serverAddr := (":8080")
	if err := e.Start(serverAddr); err != nil {
		logrus.Fatal("Error starting server: ", err)
	}
}
