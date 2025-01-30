package main

import (
	"SafeBox/config"
	"SafeBox/graph"
	"SafeBox/handlers"
	"SafeBox/jobs"
	"SafeBox/middlewares"
	"SafeBox/migrations"
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/services"
	_ "SafeBox/services/storage"
	"SafeBox/storage"
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql/playground"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	config.InitRedis()
	db := initDB()
	runMigrations(db)
	seedDatabase(db)

	p2pStorage, err := storage.NewP2PStorageAdapter(config.RedisClient)
	if err != nil {
		log.Fatalf("Failed to initialize P2P storage: %v", err)
	}

	// Inicializar storages
	baseDir := os.Getenv("STORAGE_BASE_DIR")
	if baseDir == "" {
		baseDir = "./storage"
	}

	localStorage := storage.NewLocalStorage(baseDir)
	p2pStorage := storage.NewP2PStorage()
	r2Storage := storage.NewR2Storage()
	unifiedStorage := storage.NewUnifiedStorage(localStorage, p2pStorage, r2Storage)

	// Serviços
	quotaRepo := repositories.NewQuotaRepository(db)
	quotaService := services.NewQuotaService(quotaRepo, unifiedStorage, config.RedisClient)
	quotaHandler := handlers.NewQuotaHandler(quotaService)
	quotaMiddleware := middlewares.NewQuotaMiddleware(quotaService, config.RedisClient)

	// Configurar job de reconciliação com processamento em batch
	batchProcessor := jobs.NewBatchProcessor(1000, func(userIDs []uint) error {
		ctx := context.Background()
		for _, userID := range userIDs {
			if err := quotaRepo.ReconcileUserQuota(ctx, userID); err != nil {
				log.Printf("Error reconciling user %d: %v", userID, err)
			}
		}
		return nil
	})

	go jobs.StartReconciliationJob(quotaRepo, unifiedStorage, batchProcessor)

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
// ... [initDB, runMigrations, seedDatabase, startServer]s

// Carregar variáveis de ambiente
func loadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Erro ao carregar o arquivo .env: %v", err)
	}
}

// Inicializar banco de dados
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

// Executar migrações
func runMigrations(db *gorm.DB) {
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatalf("Falha ao executar migrações: %v", err)
	}
	log.Println("Migrações executadas com sucesso!")
}

// Seed do banco de dados (permissões)
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

// Configurar OAuth2
func configureOAuth() *oauth2.Config {
	redirectURL := os.Getenv("OAUTH_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/oauth/callback" // Default local
	}

	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// Iniciar servidor
func startServer(e *echo.Echo) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor iniciado na porta :%s", port)
	log.Fatal(e.Start(":" + port))
}
