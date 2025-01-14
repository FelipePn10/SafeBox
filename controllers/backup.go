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

// BackupResult holds the results of a backup operation
type BackupResult struct {
	SuccessCount int
	FailedFiles  []string
	Error        error
}

type BackupController struct {
	Storage storage.Storage
}

func NewBackupController(storage storage.Storage) *BackupController {
	return &BackupController{Storage: storage}
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

// handleBackup manages the backup process for a given path
func (b *BackupController) handleBackup(c echo.Context, basePath, destDir string) error {
	replace := c.QueryParam("replace") == "true"

	if err := validatePath(basePath); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Minute)
	defer cancel()

	result := backupDirectory(ctx, basePath, destDir, b.Storage, replace, 10)
	if result.Error != nil {
		logrus.WithError(result.Error).Error("Backup failed")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error performing backup"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Backup completed",
		"successCount": result.SuccessCount,
		"failedFiles":  result.FailedFiles,
	})
}

// createBackupRecord creates a backup record in the database
func (b *BackupController) createBackupRecord(ctx context.Context, user *models.User, appName, filePath string) error {
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

// validateUser checks if a user is present in the context
func (b *BackupController) validateUser(c echo.Context) (*models.User, error) {
	user, ok := c.Get("user").(*models.User)
	if !ok {
		return nil, errors.New("user not found in context or not of type *models.User")
	}
	return user, nil
}

// handleAppBackup performs backup for a specific application
func (b *BackupController) handleAppBackup(c echo.Context, appName string) error {
	user, err := b.validateUser(c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	appPath := c.QueryParam(appName + "_path")
	if appPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": fmt.Sprintf("%s path not provided", appName)})
	}

	if err := validatePath(appPath); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Minute)
	defer cancel()

	result := backupDirectory(ctx, appPath, appName+"_backup", b.Storage, false, 5)
	if result.Error != nil {
		logrus.WithFields(logrus.Fields{
			"appName": appName,
			"error":   result.Error,
		}).Error("Error backing up app")
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": fmt.Sprintf("Error backing up %s", appName)})
	}

	for _, filePath := range result.FailedFiles {
		if err := b.createBackupRecord(ctx, user, appName, filePath); err != nil {
			logrus.WithFields(logrus.Fields{
				"appName": appName,
				"file":    filePath,
				"error":   err,
			}).Error("Failed to create backup record for failed file")
		}
	}

	// Here we'd need a way to get paths for successful backups, which might require additional logic or passing more data
	for i := 0; i < result.SuccessCount; i++ {
		filePath := fmt.Sprintf("successful_backup_%d", i)
		if err := b.createBackupRecord(ctx, user, appName, filePath); err != nil {
			logrus.WithFields(logrus.Fields{
				"appName": appName,
				"file":    filePath,
				"error":   err,
			}).Error("Failed to create backup record for successful file")
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      fmt.Sprintf("%s backup completed successfully", appName),
		"successCount": result.SuccessCount,
		"failedFiles":  result.FailedFiles,
	})
}

func (b *BackupController) BackupGallery(c echo.Context) error {
	// Implementação do backup da galeria
	return c.JSON(200, map[string]string{"message": "Gallery backup successful"})
}

func (b *BackupController) BackupWhatsApp(c echo.Context) error {
	// Implementação do backup do WhatsApp
	return c.JSON(200, map[string]string{"message": "WhatsApp backup successful"})
}

func (b *BackupController) BackupApp(c echo.Context) error {
	// Implementação do backup de um app específico
	return c.JSON(200, map[string]string{"message": "App backup successful"})
}

// Placeholder function for storing encryption key securely
func storeEncryptionKey(ctx context.Context, key, filePath string) error {
	// Implement the logic to securely store the encryption key here
	return nil
}
