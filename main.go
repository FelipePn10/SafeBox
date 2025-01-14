package main

import (
	"SafeBox/middlewares"
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/routes"
	"SafeBox/services"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Carrega variáveis de ambiente do arquivo .env
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Configuração do banco de dados PostgreSQL
	dsn := buildDatabaseDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Migrações
	db.AutoMigrate(&models.User{}, &models.Backup{}, &models.BackupHistory{}, &models.TempTwoFASecret{})

	// Repositórios
	userRepo := repositories.NewUserRepository(db)
	backupRepo := repositories.NewBackupRepository(db)
	twoFactorRepo := repositories.NewTwoFactorRepository(db)

	//middlewares
	authMiddleware := middlewares.NewAuthMiddleware(authService.TokenValidator, userRepo)

	// Serviços
	authService := services.NewAuthService(userRepo)
	backupService := services.NewBackupService(backupRepo)
	twoFactorService := services.NewTwoFactorService(twoFactorRepo)

	// Configuração do Echo e registro das rotas
	e := routes.NewRouteConfig(authService, backupService, twoFactorService)
	e.Start(":" + os.Getenv("PORT"))
}

// buildDatabaseDSN constrói a string de conexão do PostgreSQL a partir das variáveis de ambiente
func buildDatabaseDSN() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	return "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable TimeZone=UTC"
}
