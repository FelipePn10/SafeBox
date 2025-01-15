package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/storage"
	"SafeBox/utils"
)

type BackupResult struct {
	SuccessCount int
	FailedFiles  []string
	Error        error
}

const (
	maxFileSize      = 25 * 1024 * 1024 * 1024 // 25 GB
	maxBackupsPerDay = 10
)

type BackupConfig struct {
	AllowedExtensions []string
	MaxFileSize       int64
	BasePath          string
}

type BackupController struct {
	Storage    storage.Storage
	backupRepo *repositories.BackupRepository
}

func NewBackupController(storage storage.Storage, backupRepo *repositories.BackupRepository) *BackupController {
	return &BackupController{
		Storage:    storage,
		backupRepo: backupRepo,
	}
}

func (b *BackupController) Backup(c echo.Context) error {
	user, err := b.validateUser(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	count, err := b.backupRepo.CountUserBackupsToday(user.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to count user backups")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	if count >= maxBackupsPerDay {
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "daily backup limit exceeded"})
	}

	backupType := c.QueryParam("type")
	config, err := b.getBackupConfig(backupType)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	result, err := b.processBackup(c.Request().Context(), user, config)
	if err != nil {
		logrus.WithError(err).Error("Backup failed")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "backup failed"})
	}

	return c.JSON(http.StatusOK, result)
}

func (b *BackupController) getBackupConfig(backupType string) (*BackupConfig, error) {
	switch backupType {
	case "gallery":
		return &BackupConfig{
			AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".gif"},
			MaxFileSize:       maxFileSize,
			BasePath:          "gallery",
		}, nil
	case "documents":
		return &BackupConfig{
			AllowedExtensions: []string{".pdf", ".doc", ".docx", ".txt"},
			MaxFileSize:       maxFileSize,
			BasePath:          "documents",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported backup type: %s", backupType)
	}
}

func (b *BackupController) processBackup(ctx context.Context, user *models.OAuthUser, config *BackupConfig) (*BackupResult, error) {
	basePath := config.BasePath
	destDir := filepath.Join("backups", basePath)

	// Valida o caminho base
	if err := validatePath(basePath); err != nil {
		return nil, fmt.Errorf("invalid base path: %w", err)
	}

	// Realiza o backup do diretório
	result := backupDirectory(ctx, basePath, destDir, b.Storage, false, 10)
	if result.Error != nil {
		return nil, result.Error
	}

	// Cria registros de backup no banco de dados
	for _, filePath := range result.FailedFiles {
		if err := b.createBackupRecord(ctx, user, basePath, filePath); err != nil {
			logrus.WithFields(logrus.Fields{
				"file":  filePath,
				"error": err,
			}).Error("Failed to create backup record for failed file")
		}
	}

	for i := 0; i < result.SuccessCount; i++ {
		filePath := fmt.Sprintf("successful_backup_%d", i)
		if err := b.createBackupRecord(ctx, user, basePath, filePath); err != nil {
			logrus.WithFields(logrus.Fields{
				"file":  filePath,
				"error": err,
			}).Error("Failed to create backup record for successful file")
		}
	}

	return &result, nil
}

func (b *BackupController) validateUser(c echo.Context) (*models.OAuthUser, error) {
	user, ok := c.Get("user").(*models.OAuthUser)
	if !ok {
		return nil, errors.New("user not found in context or not of type *models.User")
	}
	return user, nil
}

// validatePath checks if the given path is valid and accessible
func validatePath(path string) error {
	if path == "" {
		return errors.New("path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("invalid or inaccessible path: %w", err)
	}
	if !info.IsDir() {
		return errors.New("the given path is not a directory")
	}
	return nil
}

// compressAndEncrypt compresses and encrypts a file
func compressAndEncrypt(filePath string) ([]byte, string, error) {
	compressedFile, err := utils.Compress(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("compression failed: %w", err)
	}

	encryptionKey, err := utils.GenerateEncryptionKey()
	if err != nil {
		return nil, "", fmt.Errorf("encryption key generation failed: %w", err)
	}
	var encryptedBuffer bytes.Buffer
	err = utils.EncryptStream(bytes.NewReader(compressedFile), &encryptedBuffer, encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("encryption failed: %w", err)
	}
	return encryptedBuffer.Bytes(), string(encryptionKey), nil
}

// processAndUpload processes a single file for backup
func processAndUpload(ctx context.Context, filePath, destPath string, storage storage.Storage, replace bool) error {
	encryptedFile, encryptionKey, err := compressAndEncrypt(filePath)
	if err != nil {
		return err
	}

	if !replace {
		exists, err := storage.Exists(destPath)
		if err != nil {
			return fmt.Errorf("failed to check if file exists: %w", err)
		}
		if exists {
			logrus.WithFields(logrus.Fields{
				"file": destPath,
			}).Warn("The file already exists. Skipping upload.")
			return nil
		}
	}

	_, err = storage.Upload(bytes.NewReader(encryptedFile), destPath)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	err = storeEncryptionKey(ctx, encryptionKey, destPath)
	if err != nil {
		return fmt.Errorf("failed to store encryption key: %w", err)
	}

	return nil
}

// backupDirectory backups a directory, processing files concurrently
func backupDirectory(ctx context.Context, basePath, destDir string, storage storage.Storage, replace bool, maxWorkers int) BackupResult {
	var (
		wg            sync.WaitGroup
		mu            sync.Mutex
		failedFiles   = make(chan string, 100)
		expectedCount = 0
	)

	sem := semaphore.NewWeighted(int64(maxWorkers))

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk error at %s: %w", path, err)
		}
		if info.IsDir() {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			defer sem.Release(1)

			relPath, err := filepath.Rel(basePath, filePath)
			if err != nil {
				failedFiles <- filePath
				return
			}

			destPath := filepath.Join(destDir, relPath)
			if err := processAndUpload(ctx, filePath, destPath, storage, replace); err != nil {
				failedFiles <- filePath
			} else {
				mu.Lock()
				expectedCount++
				mu.Unlock()
			}
		}(path)
		return nil
	})

	go func() {
		wg.Wait()
		close(failedFiles)
	}()

	var failed []string
	for f := range failedFiles {
		failed = append(failed, f)
	}

	return BackupResult{
		SuccessCount: expectedCount,
		FailedFiles:  failed,
		Error:        err,
	}
}

// createBackupRecord creates a backup record in the database
func (b *BackupController) createBackupRecord(ctx context.Context, user *models.OAuthUser, appName, filePath string) error {
	backup := models.Backup{
		UserID:   user.ID,
		AppName:  appName,
		FilePath: filePath,
	}

	backupHistory := models.BackupHistory{
		UserID:     user.ID,
		AppName:    appName,
		BackupDate: time.Now(),
		BackupMode: "manual",
		FilePath:   filePath,
	}

	tx := repositories.DBConnection.WithContext(ctx).Begin()
	if err := tx.Create(&backup).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create backup record: %w", err)
	}

	if err := tx.Create(&backupHistory).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create backup history: %w", err)
	}

	return tx.Commit().Error
}

// storeEncryptionKey stores the encryption key securely in the database
func storeEncryptionKey(ctx context.Context, key, filePath string) error {
	encryptionKeyRecord := models.EncryptionKey{
		FilePath:  filePath,
		Key:       key,
		CreatedAt: time.Now(),
	}

	// Salva a chave no banco de dados
	if err := repositories.DBConnection.WithContext(ctx).Create(&encryptionKeyRecord).Error; err != nil {
		return fmt.Errorf("failed to store encryption key: %w", err)
	}

	return nil
}
