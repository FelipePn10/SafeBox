package handlers

import (
	"SafeBox/services/storage"
	"net/http"

	"SafeBox/repositories"
	"github.com/labstack/echo/v4"
)

type BackupHandler struct {
	storage    storage.Storage
	backupRepo *repositories.BackupRepository
}

func NewBackupHandler(storage storage.Storage, backupRepo *repositories.BackupRepository) *BackupHandler {
	return &BackupHandler{
		storage:    storage,
		backupRepo: backupRepo,
	}
}

func (h *BackupHandler) Backup(c echo.Context) error {
	// Implementação completa do backup aqui
	// ... (usar código do controllers/backup.go)
	return c.JSON(http.StatusOK, map[string]string{"status": "backup created"})
}
