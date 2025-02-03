package main

import (
	"SafeBox/config"
	"SafeBox/graph"
	"SafeBox/handlers"
	jobs "SafeBox/job"
	"SafeBox/middlewares"
	"SafeBox/migrations"
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/services"
	"SafeBox/services/storage"
	_ "context"
	"fmt"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "golang.org/x/oauth2"
	_ "golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	config.InitRedis()
	db := initDB()
	runMigrations(db)
	seedDatabase(db)

	// Inicializar storages
	baseDir := os.Getenv("STORAGE_BASE_DIR")
	if baseDir == "" {
		baseDir = "./storage"
	}
	localStorage := storage.NewLocalStorage(baseDir)
	p2pStorage := storage.NewP2PStorage(config.RedisClient) // Corrigindo a chamada
	r2Storage, _ := storage.NewR2Storage()
	unifiedStorage := storage.NewUnifiedStorage(localStorage, p2pStorage, r2Storage)

	// Serviços
	quotaRepo := repositories.NewQuotaRepository(db)
	quotaService := services.NewQuotaService(quotaRepo, unifiedStorage, config.RedisClient)
	quotaHandler := handlers.NewQuotaHandler(quotaService)
	quotaMiddleware := middlewares.NewQuotaMiddleware(quotaService, config.RedisClient)

	// Configurar job de reconciliação com processamento em batch
	go jobs.StartReconciliationJob(quotaRepo, unifiedStorage)

	// Echo
	e := echo.New()
	e.Use(
		middleware.Logger(),
		middleware.Recover(),
		middleware.CORS(),
		quotaMiddleware.EnforceQuota,
	)

	// Rotas
	e.GET("/api/quota", quotaHandler.GetQuotaUsage)

	// GraphQL
	srv := graph.NewGraphQLHandler(db)
	e.GET("/playground", echo.WrapHandler(playground.Handler("GraphQL Playground", "/query")))
	e.POST("/query", echo.WrapHandler(srv))

	startServer(e)
}

// Funções auxiliares (mantidas sem alterações)
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
		log.Fatalf("Falha ao conectar no banco de dados: %v", err)
	}
	return db
}

func runMigrations(db *gorm.DB) {
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatalf("Falha ao executar migrações: %v", err)
	}
	log.Println("Migrações executadas com sucesso!")
}

func seedDatabase(db *gorm.DB) {
	permissions := []models.PermissionModel{
		{Name: string(models.READ)},
		{Name: string(models.WRITE)},
		{Name: string(models.DELETE)},
		{Name: string(models.ADMIN)},
		{Name: string(models.PermissionBackup)},
	}
	for _, p := range permissions {
		if err := db.FirstOrCreate(&p, "name = ?", p.Name).Error; err != nil {
			log.Fatalf("Falha ao seedar permissão %s: %v", p.Name, err)
		}
	}
	log.Println("Permissões seedadas com sucesso!")
}

func startServer(e *echo.Echo) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor iniciado na porta :%s", port)
	log.Fatal(e.Start(":" + port))
}
