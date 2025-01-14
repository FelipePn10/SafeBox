package controllers

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/storage"
	"SafeBox/utils"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/labstack/echo/v4"
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
func (f *FileController) Upload(c echo.Context) error {
	logrus.Info("Recebendo solicitação de upload de arquivo")
	uploadCounter.Inc()

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "File not found or invalid"})
	}
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error opening file"})
	}
	defer src.Close()

	// Verificar limite de armazenamento
	user := c.Get("user").(*models.User)
	if user.Plan == "free" && user.StorageUsed+file.Size > user.StorageLimit {
		return c.JSON(http.StatusForbidden, map[string]interface{}{"error": "Storage limit exceeded"})
	}

	// Criptografar arquivo
	encryptionKey, err := utils.GenerateEncryptionKey()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error generating encryption key"})
	}
	var encryptedFile bytes.Buffer
	if err := utils.EncryptStream(src, &encryptedFile, encryptionKey); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error encrypting the file"})
	}

	// Salvar arquivo criptografado
	path, err := f.Storage.Upload(bytes.NewReader(encryptedFile.Bytes()), file.Filename)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error saving the file"})
	}

	// Atualizar espaço de armazenamento usado
	user.StorageUsed += file.Size
	// Salvar usuário atualizado no banco de dados
	if err := repositories.NewUserRepository(repositories.DBConnection).Update(user); err != nil {
		logrus.Error("Erro ao atualizar espaço de armazenamento usado: ", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error updating storage usage"})
	}

	// Enviar notificação por e-mail no primeiro upload
	if user.StorageUsed == file.Size {
		if err := utils.SendEmail(user.Email, "Primeiro Upload Realizado", "Parabéns! Você realizou seu primeiro upload."); err != nil {
			logrus.Error("Erro ao enviar e-mail de notificação: ", err)
		}
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "File uploaded successfully",
		"path":    path,
	})
}

// Download function to handle file download
func (f *FileController) Download(c echo.Context) error {
	logrus.Info("Recebendo solicitação de download de arquivo")
	downloadCounter.Inc()

	filename := c.Param("id")
	file, err := f.Storage.Download(filename)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "File not found"})
	}
	defer file.Close()

	// Ler o conteúdo do arquivo
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error reading the file"})
	}

	// Descriptografar arquivo
	encryptionKey, err := utils.GenerateEncryptionKey() // Recuperar a chave de criptografia correta
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error generating encryption key"})
	}
	var decryptedFile bytes.Buffer
	err = utils.DecryptStream(bytes.NewReader(fileContent), &decryptedFile, encryptionKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error decrypting the file"})
	}

	return c.Blob(http.StatusOK, "application/octet-stream", decryptedFile.Bytes())
}

// Delete function to handle file deletion
func (f *FileController) Delete(c echo.Context) error {
	logrus.Info("Recebendo solicitação de exclusão de arquivo")
	deleteCounter.Inc()

	filename := c.Param("id")
	if err := f.Storage.Delete(filename); err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "File not found"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "File deleted"})
}

// ListFiles function to list all uploaded files
func (f *FileController) ListFiles(c echo.Context) error {
	logrus.Info("Recebendo solicitação de listagem de arquivos")
	files, err := ioutil.ReadDir("./uploads")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error reading directory"})
	}
	var fileList []string
	for _, file := range files {
		fileList = append(fileList, file.Name()) // Append each file name to the list
	}
	return c.JSON(http.StatusOK, fileList)
}

// Update function to handle file updates (replace an existing file)
func (f *FileController) Update(c echo.Context) error {
	logrus.Info("Recebendo solicitação de atualização de arquivo")
	id := c.Param("id")
	filePath := "./uploads/" + id

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "File not found"})
	}

	// Get the new file and header to replace the old one
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "File not found"})
	}

	if file.Size > 15*1024*1024*1024 {
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]interface{}{"error": "File size exceeds the limit of 15GB"})
	}

	allowed := []string{"image/jpeg", "image/png", "application/pdf", "application/zip", "application/x-rar-compressed", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation", "text/plain"}
	if !contains(allowed, file.Header.Get("Content-Type")) {
		return c.JSON(http.StatusUnsupportedMediaType, map[string]interface{}{"error": "File type not allowed"})
	}

	// Remove the old file
	err = os.Remove(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error deleting the old file"})
	}

	// Save the new file using the header information
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error opening the new file"})
	}
	defer src.Close()
	dst, err := os.Create(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error creating new file"})
	}
	defer dst.Close()
	if _, err = io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Error writing new file"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "File updated"})
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
