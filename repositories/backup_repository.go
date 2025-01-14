package repositories

import (
	"SafeBox/models"

	"gorm.io/gorm"
)

type BackupRepository struct {
	db *gorm.DB
}

func NewBackupRepository(db *gorm.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

func (r *BackupRepository) CreateBackup(backup *models.Backup) error {
	return r.db.Create(backup).Error
}

func (r *BackupRepository) CreateBackupHistory(history *models.BackupHistory) error {
	return r.db.Create(history).Error
}
