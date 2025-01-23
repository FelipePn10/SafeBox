package main

import (
	"SafeBox/migrations"
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/routes"
	"SafeBox/services"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Initialize database
	db := initDB()

	// Run migrations
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed permissions
	if err := seedPermissions(db); err != nil {
		log.Fatalf("Failed to seed permissions: %v", err)
	}

	// Configure OAuth
	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OAUTH_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	// Initialize repositories and services
	userRepo := repositories.NewUserRepository(db)
	backupRepo := repositories.NewBackupRepository(db)
	authService := services.NewAuthService(userRepo, oauthConfig)
	backupService := services.NewBackupService(backupRepo)

	// Setup routes
	e, err := routes.NewRouteConfig(authService, backupService, db, oauthConfig)
	if err != nil {
		log.Fatalf("Failed to configure routes: %v", err)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s", port)
	log.Fatal(e.Start(":" + port))
}

func initDB() *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

func seedPermissions(db *gorm.DB) error {
	permissions := []models.PermissionModel{
		{Name: string(models.READ)},
		{Name: string(models.WRITE)},
		{Name: string(models.DELETE)},
		{Name: string(models.ADMIN)},
		{Name: string(models.PermissionBackup)},
	}

	for _, p := range permissions {
		result := db.FirstOrCreate(&p, "name = ?", p.Name)
		if result.Error != nil {
			return fmt.Errorf("failed to seed permission %s: %w", p.Name, result.Error)
		}
	}
	log.Println("Permissions seeded successfully")
	return nil
}
