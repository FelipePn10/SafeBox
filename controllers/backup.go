package controllers

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/storage"
	"SafeBox/utils"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

type BackupController struct {
	Storage storage.Storage
}

func NewBackupController(storage storage.Storage) *BackupController {
	if storage == nil {
		panic("storage cannot be nil")
	}
	return &BackupController{Storage: storage}
}

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

func compressAndEncrypt(filePath string) ([]byte, string, error) {
	compressedFile, err := utils.Compress(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("compression failed: %w", err)
	}

	encryptionKey := utils.GenerateEncryptionKey()
	encryptedFile, err := utils.EncryptFile(bytes.NewReader(compressedFile), encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("encryption failed: %w", err)
	}

	return encryptedFile, encryptionKey, nil
}

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

	// Store encryption key securely - implement this based on your security requirements
	err = storeEncryptionKey(ctx, encryptionKey, destPath)
	if err != nil {
		return fmt.Errorf("failed to store encryption key: %w", err)
	}

	return nil
}

type BackupResult struct {
	SuccessCount int
	FailedFiles  []string
	Error        error
}

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

func (b *BackupController) handleBackup(c *gin.Context, basePath, destDir string) {
	logrus.WithFields(logrus.Fields{
		"basePath": basePath,
		"destDir":  destDir,
	}).Info("Starting backup")

	if err := validatePath(basePath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	replace := c.Query("replace") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	result := backupDirectory(ctx, basePath, destDir, b.Storage, replace, 10)
	if result.Error != nil {
		logrus.WithError(result.Error).Error("Backup failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error performing backup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Backup completed",
		"successCount": result.SuccessCount,
		"failedFiles":  result.FailedFiles,
	})
}

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

	tx := repositories.DBConection.WithContext(ctx).Begin()
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

func (b *BackupController) validateUser(c *gin.Context) (*models.User, error) {
	user, ok := c.Get("user")
	if !ok {
		return nil, errors.New("user not found in context")
	}

	userModel, ok := user.(*models.User)
	if !ok {
		return nil, errors.New("user is not of type *models.User")
	}

	return userModel, nil
}

func (b *BackupController) handleAppBackup(c *gin.Context, appName string) {
	user, err := b.validateUser(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	appPath := c.Query(appName + "_path")
	if appPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s path not provided", appName)})
		return
	}

	if err := validatePath(appPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	result := backupDirectory(ctx, appPath, appName+"_backup", b.Storage, false, 5)
	if result.Error != nil {
		logrus.WithFields(logrus.Fields{
			"appName": appName,
			"error":   result.Error,
		}).Error("Error backing up app")
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error backing up %s", appName)})
		return
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

	for i := 0; i < result.SuccessCount; i++ {
		// Assuming we have a way to get the file path for successful backups, for example from a list
		filePath := fmt.Sprintf("successful_backup_%d", i)
		if err := b.createBackupRecord(ctx, user, appName, filePath); err != nil {
			logrus.WithFields(logrus.Fields{
				"appName": appName,
				"file":    filePath,
				"error":   err,
			}).Error("Failed to create backup record for successful file")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("%s backup completed successfully", appName),
		"successCount": result.SuccessCount,
		"failedFiles":  result.FailedFiles,
	})
}

func (b *BackupController) BackupGallery(c *gin.Context) {
	b.handleAppBackup(c, "gallery")
}

func (b *BackupController) BackupWhatsApp(c *gin.Context) {
	b.handleAppBackup(c, "whatsapp")
}

func (b *BackupController) BackupApp(c *gin.Context) {
	appName := c.Query("app_name")
	if appName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App name not provided"})
		return
	}
	b.handleAppBackup(c, appName)
}

// Placeholder function for storing encryption key securely
func storeEncryptionKey(ctx context.Context, key, filePath string) error {
	// Implement the logic to securely store the encryption key here
	// This is just a placeholder function
	return nil
}
