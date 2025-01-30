package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) *LocalStorage {
	return &LocalStorage{baseDir: baseDir}
}

func (ls *LocalStorage) Save(ctx context.Context, file io.Reader, userID uint, fileName string) error {
	userDir := filepath.Join(ls.baseDir, fmt.Sprintf("user_%d", userID))
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	filePath := filepath.Join(userDir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, file); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (ls *LocalStorage) GetTotalUsage(ctx context.Context, userID uint) (int64, error) {
	var total int64
	userDir := filepath.Join(ls.baseDir, fmt.Sprintf("user_%d", userID))

	err := filepath.Walk(userDir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})

	if os.IsNotExist(err) {
		return 0, nil
	}
	return total, err
}

func (ls *LocalStorage) Delete(ctx context.Context, userID uint, fileName string) error {
	filePath := filepath.Join(ls.baseDir, fmt.Sprintf("user_%d", userID), fileName)
	return os.Remove(filePath)
}
