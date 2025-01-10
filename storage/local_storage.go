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
	logrus.Infof("Iniciando upload do arquivo: %s", filename)
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
	logrus.Infof("Iniciando download do arquivo: %s", filename)
	path := filepath.Join(s.BasePath, filename)
	return os.Open(path)
}

func (s *LocalStorage) Delete(filename string) error {
	logrus.Infof("Iniciando exclusão do arquivo: %s", filename)
	path := filepath.Join(s.BasePath, filename)
	return os.Remove(path)
}
