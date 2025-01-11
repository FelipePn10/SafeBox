package storage

import (
	"io"
	"os"
)

type Storage interface {
	Upload(file io.Reader, filename string) (string, error)
	Download(filename string) (*os.File, error)
	Delete(filename string) error
	Exists(filePath string) (bool, error)
}
