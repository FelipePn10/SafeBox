package main

import (
	"SafeBox/graph"
	"SafeBox/migrations"
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/routes"
	"SafeBox/services"
	"fmt"
	"log"
	_ "net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Carregar variáveis de ambiente
	loadEnv()

	// Inicializar banco de dados
	db := initDB()

	// Executar migrações
	runMigrations(db)

	// Seed das permissões
	seedDatabase(db)

	// Configurar OAuth2
	oauthConfig := configureOAuth()

	// Inicializar repositórios e serviços
	userRepo := repositories.NewUserRepository(db)
	backupRepo := repositories.NewBackupRepository(db)
	authService := services.NewAuthService(userRepo, oauthConfig)
	backupService := services.NewBackupService(backupRepo)

	// Configurar GraphQL
	resolver := &graph.Resolver{
		DB: db,
	}
	srv := handler.NewDefaultServer(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: resolver,
			},
		),
	)

	// Configurar servidor Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Configurar rotas REST
	// Aqui mudamos para usar NewRouteConfig em vez de ConfigureRoutes
	routeConfig, err := routes.NewRouteConfig(authService, backupService, db, oauthConfig)
	if err != nil {
		log.Fatalf("Erro ao configurar rotas: %v", err)
	}
	e = routeConfig

	// Configurar rotas GraphQL
	e.GET("/playground", echo.WrapHandler(playground.Handler("GraphQL playground", "/query")))
	e.POST("/query", echo.WrapHandler(srv))

	// Iniciar servidor
	startServer(e)
}

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
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OAUTH_REDIRECT_URL"),
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
