package storage

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type LocalStorage struct {
	BasePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{BasePath: basePath}
}

func (s *LocalStorage) Upload(file io.Reader, filename string) (string, error) {
	logrus.Infof("Starting file upload: %s", filename)
	path := filepath.Join(s.BasePath, filename)
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}
	return path, nil
}

func (s *LocalStorage) Download(filename string) (*os.File, error) {
	logrus.Infof("Starting file download: %s", filename)
	path := filepath.Join(s.BasePath, filename)
	return os.Open(path)
}

func (s *LocalStorage) Delete(filename string) error {
	logrus.Infof("Initiating file deletion: %s", filename)
	path := filepath.Join(s.BasePath, filename)
	return os.Remove(path)
}

func (ls *LocalStorage) Exists(filePath string) (bool, error) {

	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {

		return false, nil

	}

	return err == nil, err

}
