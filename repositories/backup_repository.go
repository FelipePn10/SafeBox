package repositories

import (
	"SafeBox/models"
	"time"

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

func (r *BackupRepository) CountUserBackupsToday(userID uint) (int64, error) {
	var count int64
	startOfDay := time.Now().Truncate(24 * time.Hour)
	err := r.db.Model(&models.Backup{}).Where("user_id = ? AND created_at >= ?", userID, startOfDay).Count(&count).Error
	return count, err
}
