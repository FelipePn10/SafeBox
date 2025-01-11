package controllers

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/storage"
	"SafeBox/utils"
	"bytes"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	downloadCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "download_requests_total",
			Help: "Total number of download requests",
		},
	)
	uploadCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "upload_requests_total",
			Help: "Total number of upload requests",
		},
	)
	deleteCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "delete_requests_total",
			Help: "Total number of delete requests",
		},
	)
)

type FileController struct {
	Storage storage.Storage
}

// NewFileController creates a new instance of FileController
func NewFileController(storage storage.Storage) *FileController {
	return &FileController{Storage: storage}
}

// Upload function to handle file upload
func (f *FileController) Upload(c *gin.Context) {
	logrus.Info("Recebendo solicitação de upload de arquivo")
	uploadCounter.Inc()

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File not found or invalid"})
		return
	}
	defer file.Close()

	// Verificar limite de armazenamento
	user := c.MustGet("user").(*models.User)
	if user.Plan == "free" && user.StorageUsed+header.Size > user.StorageLimit {
		c.JSON(http.StatusForbidden, gin.H{"error": "Storage limit exceeded"})
		return
	}

	// Criptografar arquivo
	encryptionKey := utils.GenerateEncryptionKey()
	encryptedFile, err := utils.EncryptFile(file, encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error encrypting the file"})
		return
	}

	// Salvar arquivo criptografado
	path, err := f.Storage.Upload(bytes.NewReader(encryptedFile), header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving the file"})
		return
	}

	// Atualizar espaço de armazenamento usado
	user.StorageUsed += header.Size
	// Salvar usuário atualizado no banco de dados
	if err := repositories.NewUserRepository(repositories.DBConection).Update(user); err != nil {
		logrus.Error("Erro ao atualizar espaço de armazenamento usado: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating storage usage"})
		return
	}

	// Enviar notificação por e-mail no primeiro upload
	if user.StorageUsed == header.Size {
		if err := utils.SendEmail(user.Email, "Primeiro Upload Realizado", "Parabéns! Você realizou seu primeiro upload."); err != nil {
			logrus.Error("Erro ao enviar e-mail de notificação: ", err)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "File uploaded successfully",
		"path":    path,
	})
}

// Download function to handle file download
func (f *FileController) Download(c *gin.Context) {
	logrus.Info("Recebendo solicitação de download de arquivo")
	downloadCounter.Inc()

	filename := c.Param("id")
	file, err := f.Storage.Download(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	defer file.Close()

	// Ler o conteúdo do arquivo
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading the file"})
		return
	}

	// Descriptografar arquivo
	encryptionKey := utils.GenerateEncryptionKey() // Recuperar a chave de criptografia correta
	decryptedFile, err := utils.DecryptFile(fileContent, encryptionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decrypting the file"})
		return
	}

	c.Data(http.StatusOK, "application/octet-stream", decryptedFile)
}

func (f *FileController) Delete(c *gin.Context) {
	logrus.Info("Recebendo solicitação de exclusão de arquivo")
	deleteCounter.Inc()

	filename := c.Param("id")
	if err := f.Storage.Delete(filename); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted"})
}

// ListFiles function to list all uploaded files
func (f *FileController) ListFiles(c *gin.Context) {
	logrus.Info("Recebendo solicitação de listagem de arquivos")
	files, err := ioutil.ReadDir("./uploads")
	if err != nil {
		c.JSON(500, gin.H{"error": "Error reading directory"})
		return
	}
	var fileList []string
	for _, file := range files {
		fileList = append(fileList, file.Name()) // Append each file name to the list
	}
	c.JSON(200, fileList)
}

// Update function to handle file updates (replace an existing file)
func (f *FileController) Update(c *gin.Context) {
	logrus.Info("Recebendo solicitação de atualização de arquivo")
	id := c.Param("id")
	filePath := "./uploads/" + id

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	// Get the new file and header to replace the old one
	_, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "File not found"})
		return
	}

	if header.Size > 15*1024*1024*1024 {
		c.JSON(413, gin.H{"error": "File size exceeds the limit of 10GB"})
		return
	}

	allowed := []string{"image/jpeg", "image/png", "application/pdf", "application/zip", "application/x-rar-compressed", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "text/plain"}
	if !contains(allowed, header.Header.Get("Content-Type")) {
		c.JSON(415, gin.H{"error": "File type not allowed"})
		return
	}

	// Remove the old file
	err = os.Remove(filePath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error deleting the old file"})
		return
	}

	// Save the new file using the header information
	err = c.SaveUploadedFile(header, filePath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error saving the new file"})
		return
	}

	c.JSON(200, gin.H{"message": "File updated"})
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
