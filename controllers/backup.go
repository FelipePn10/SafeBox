package controllers

import (
	"SafeBox/storage"
	"SafeBox/utils"
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type BackupController struct {
	Storage storage.Storage
}

// NewBackupController creates a new instance of BackupController
func NewBackupController(storage storage.Storage) *BackupController {
	return &BackupController{Storage: storage}
}

// validatePath validates that the given path is a valid directory.
func validatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("invalid or inaccessible path")
	}
	if !info.IsDir() {
		return errors.New("the given path is not a directory")
	}
	return nil
}

// compressAndEncrypt compresses and encrypts a file or directory.
func compressAndEncrypt(filePath string) ([]byte, error) {
	compressedFile, err := utils.Compress(filePath)
	if err != nil {
		return nil, err
	}

	encryptionKey := utils.GenerateEncryptionKey()
	encryptedFile, err := utils.EncryptFile(bytes.NewReader(compressedFile), encryptionKey)
	if err != nil {
		return nil, err
	}

	return encryptedFile, nil
}

// processAndUpload compresses, encrypts, and uploads the file to storage.
func processAndUpload(filePath, destPath string, storage storage.Storage, replace bool) error {
	// Compression and encryption
	encryptedFile, err := compressAndEncrypt(filePath)
	if err != nil {
		return err
	}

	// Checks whether to overwrite the existing backup
	if !replace {
		exists, _ := storage.Exists(destPath)
		if exists {
			logrus.Warnf("The file %s already exists. Skipping upload.", destPath)
			return nil
		}
	}

	// Upload the file
	_, err = storage.Upload(bytes.NewReader(encryptedFile), destPath)
	return err
}

// backupDirectory backs up a directory, processing files in parallel.
func backupDirectory(basePath, destDir string, storage storage.Storage, replace bool) (int, []string, error) {
	var wg sync.WaitGroup
	errChan := make(chan string, 100)
	successCount := 0
	mu := &sync.Mutex{}

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			destPath := filepath.Join(destDir, info.Name())
			if err := processAndUpload(filePath, destPath, storage, replace); err != nil {
				errChan <- filePath
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(path)
		return nil
	})

	wg.Wait()
	close(errChan)

	var failedFiles []string
	for failed := range errChan {
		failedFiles = append(failedFiles, failed)
	}

	return successCount, failedFiles, err
}

// askReplace asks the user whether the backup should be incremental or replace the old one.
func askReplace(c *gin.Context) bool {
	replace := c.Query("replace")
	if replace == "true" {
		return true
	}
	return false
}

// handleBackup is a generic function to perform backups of different types of data.
func (b *BackupController) handleBackup(c *gin.Context, basePath, destDir string) {
	logrus.Infof("Starting backup of: %s", basePath)

	if basePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path not provided"})
		return
	}

	if err := validatePath(basePath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	replace := askReplace(c)

	successCount, failedFiles, err := backupDirectory(basePath, destDir, b.Storage, replace)
	if err != nil {
		logrus.Error("Error during backup: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error performing backup"})
		return
	}

	response := gin.H{
		"message":      "Backup completed",
		"successCount": successCount,
		"failedFiles":  failedFiles,
	}
	c.JSON(http.StatusOK, response)
}

// BackupGallery backs up the user's photo gallery.
func (b *BackupController) BackupGallery(c *gin.Context) {
	galleryPath := c.Query("gallery_path")
	b.handleBackup(c, galleryPath, "gallery_backup")
}

// BackupWhatsApp backs up the user's WhatsApp conversations.
func (b *BackupController) BackupWhatsApp(c *gin.Context) {
	whatsappPath := c.Query("whatsapp_path")
	b.handleBackup(c, whatsappPath, "whatsapp_backup")
}

// BackupApp backs up data from a specific application.
func (b *BackupController) BackupApp(c *gin.Context) {
	appPath := c.Query("app_path")
	appName := c.Query("app_name")

	if appName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Application name not provided"})
		return
	}

	b.handleBackup(c, appPath, appName+"_backup")
}
