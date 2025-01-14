package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"io"
	"os"
)

type BackupService struct {
	backupRepo *repositories.BackupRepository
}

// Delete implements storage.Storage.
func (s *BackupService) Delete(filename string) error {
	panic("unimplemented")
}

// Download implements storage.Storage.
func (s *BackupService) Download(filename string) (*os.File, error) {
	panic("unimplemented")
}

// Exists implements storage.Storage.
func (s *BackupService) Exists(filePath string) (bool, error) {
	panic("unimplemented")
}

// Upload implements storage.Storage.
func (s *BackupService) Upload(file io.Reader, filename string) (string, error) {
	panic("unimplemented")
}

func NewBackupService(backupRepo *repositories.BackupRepository) *BackupService {
	return &BackupService{backupRepo: backupRepo}
}

func (s *BackupService) CreateBackup(backup *models.Backup) error {
	return s.backupRepo.CreateBackup(backup)
}

func (s *BackupService) CreateBackupHistory(history *models.BackupHistory) error {
	return s.backupRepo.CreateBackupHistory(history)
}
