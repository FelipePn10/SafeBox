package migrations

import (
	"SafeBox/models"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// RunMigrations executa todas as migrações necessárias
func RunMigrations(db *gorm.DB) error {
	log.Println("Running migrations...")

	// Cria a tabela de permissões
	if err := db.AutoMigrate(&models.PermissionModel{}); err != nil {
		return fmt.Errorf("failed to migrate PermissionModel: %w", err)
	}

	// Cria a tabela de usuários OAuth
	if err := db.AutoMigrate(&models.OAuthUser{}); err != nil {
		return fmt.Errorf("failed to migrate OAuthUser: %w", err)
	}

	// Cria a tabela de backups
	if err := db.AutoMigrate(&models.Backup{}); err != nil {
		return fmt.Errorf("failed to migrate Backup: %w", err)
	}

	// Cria a tabela de histórico de backups
	if err := db.AutoMigrate(&models.BackupHistory{}); err != nil {
		return fmt.Errorf("failed to migrate BackupHistory: %w", err)
	}

	// Cria a tabela de chaves de criptografia
	if err := db.AutoMigrate(&models.EncryptionKey{}); err != nil {
		return fmt.Errorf("failed to migrate EncryptionKey: %w", err)
	}

	log.Println("Migrations completed successfully!")
	return nil
}
