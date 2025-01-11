package routes

import (
	"SafeBox/controllers"
	"SafeBox/middlewares"
	"SafeBox/models"
	"SafeBox/storage"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// setupStorage configura o armazenamento S3
func setupStorage() (storage.Storage, error) {
	bucketName := os.Getenv("R2_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("variável de ambiente R2_BUCKET_NAME não configurada")
	}

	s3Storage, err := storage.NewR2Storage(bucketName)
	if err != nil {
		return nil, fmt.Errorf("falha ao configurar armazenamento S3: %w", err)
	}

	return s3Storage, nil
}

// SetupRouter configura as rotas do servidor
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Configurar armazenamento
	s3Storage, err := setupStorage()
	if err != nil {
		log.Fatalf("Erro crítico: %v", err)
	}

	backupController := controllers.NewBackupController(s3Storage)

	backup := r.Group("/backup", middlewares.Authorize(models.PermissionBackup))
	{
		backup.POST("/gallery", backupController.BackupGallery)
		backup.POST("/whatsapp", backupController.BackupWhatsApp)
		backup.POST("/app", backupController.BackupApp)
	}

	// Retornar instância do roteador
	return r
}
