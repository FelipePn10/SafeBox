package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/redis/go-redis/v9"
)

type DeduplicationService struct {
	redisClient *redis.Client
	mutex       sync.RWMutex
}

func NewDeduplicationService(redisClient *redis.Client) *DeduplicationService {
	return &DeduplicationService{
		redisClient: redisClient,
	}
}

func (ds *DeduplicationService) ProcessFile(ctx context.Context, file io.Reader) (string, []byte, error) {
	// Ler e calcular hash do arquivo
	hash := sha256.New()
	content, err := io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	hash.Write(content)
	fileHash := hex.EncodeToString(hash.Sum(nil))

	return fileHash, content, nil
}

func (ds *DeduplicationService) LinkFileToUser(ctx context.Context, fileHash string, userID uint) error {
	key := fmt.Sprintf("file_users:%s", fileHash)
	return ds.redisClient.SAdd(ctx, key, userID).Err()
}

func (ds *DeduplicationService) IsFileDuplicate(ctx context.Context, fileHash string) (bool, error) {
	exists, err := ds.redisClient.Exists(ctx, fmt.Sprintf("file:%s", fileHash)).Result()
	return exists > 0, err
}
