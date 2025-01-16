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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Carrega as variáveis de ambiente do arquivo .env
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Conecta ao banco de dados
	dsn := buildDatabaseDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Executa as migrações
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Popula a tabela de permissões com as permissões padrão
	if err := seedPermissions(db); err != nil {
		log.Fatalf("Failed to seed permissions: %v", err)
	}

	// Inicializa os repositórios
	userRepo := repositories.NewUserRepository(db)
	backupRepo := repositories.NewBackupRepository(db)

	// Inicializa os serviços
	authService := services.NewAuthService(userRepo)
	backupService := services.NewBackupService(backupRepo)

	// Configura as rotas
	e, err := routes.NewRouteConfig(authService, backupService, db)
	if err != nil {
		log.Fatalf("Failed to configure routes: %v", err)
	}

	// Inicia o servidor
	if err := e.Start(":" + os.Getenv("PORT")); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// buildDatabaseDSN constrói a string de conexão com o banco de dados
func buildDatabaseDSN() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	return "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable TimeZone=UTC"
}

// seedPermissions popula a tabela de permissões com as permissões padrão
func seedPermissions(db *gorm.DB) error {
	permissions := []models.PermissionModel{
		{Name: string(models.READ)},
		{Name: string(models.WRITE)},
		{Name: string(models.DELETE)},
		{Name: string(models.ADMIN)},
		{Name: string(models.PermissionBackup)},
	}

	for _, permission := range permissions {
		// Usa FirstOrCreate para evitar duplicação
		if err := db.FirstOrCreate(&permission, models.PermissionModel{Name: permission.Name}).Error; err != nil {
			return fmt.Errorf("failed to seed permission %s: %w", permission.Name, err)
		}
	}

	log.Println("Permissions seeded successfully!")
	return nil
}
