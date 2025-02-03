package storage

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type P2PStorageAdapter struct {
	redisClient *redis.Client
	cacheMutex  sync.RWMutex
}

func NewP2PStorageAdapter(redisClient *redis.Client) *P2PStorageAdapter {
	return &P2PStorageAdapter{
		redisClient: redisClient,
	}
}

func (pa *P2PStorageAdapter) Save(ctx context.Context, file io.Reader, userID uint, fileName string) error {
	// Criar chave única para o arquivo
	fileKey := fmt.Sprintf("%d:%s", userID, fileName)

	// Ler o conteúdo do arquivo
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Salvar no P2P
	if err := pa.p2pClient.Store(fileKey, data); err != nil {
		return fmt.Errorf("failed to store in p2p network: %w", err)
	}

	// Atualizar cache com metadados
	metadata := map[string]interface{}{
		"size":      len(data),
		"timestamp": time.Now().Unix(),
		"userId":    userID,
	}

	cacheKey := fmt.Sprintf("p2p:file:%s", fileKey)
	if err := pa.redisCache.HMSet(ctx, cacheKey, metadata).Err(); err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}

	// Adicionar à lista de arquivos do usuário
	userFilesKey := fmt.Sprintf("p2p:user:%d:files", userID)
	if err := pa.redisClient.SAdd(ctx, userFilesKey, fileKey).Err(); err != nil {
		return fmt.Errorf("failed to update user files: %w", err)
	}

	return nil
}

func (pa *P2PStorageAdapter) GetTotalUsage(ctx context.Context, userID uint) (int64, error) {
	var total int64
	userFilesKey := fmt.Sprintf("p2p:user:%d:files", userID)

	// Obter lista de arquivos do usuário
	fileKeys, err := pa.redisCache.SMembers(ctx, userFilesKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get user files: %w", err)
	}

	// Somar tamanho de cada arquivo
	for _, fileKey := range fileKeys {
		cacheKey := fmt.Sprintf("p2p:file:%s", fileKey)
		size, err := pa.redisCache.HGet(ctx, cacheKey, "size").Int64()
		if err != nil {
			// Se não encontrar no cache, tenta buscar do P2P
			data, err := pa.p2pClient.Retrieve(fileKey)
			if err != nil {
				continue // Skip if file not found
			}
			size = int64(len(data))

			// Atualizar cache
			pa.redisCache.HSet(ctx, cacheKey, "size", size)
		}
		total += size
	}

	return total, nil
}

func (pa *P2PStorageAdapter) Delete(ctx context.Context, userID uint, fileName string) error {
	fileKey := fmt.Sprintf("%d:%s", userID, fileName)

	// Remover do P2P
	if err := pa.p2pClient.Delete(fileKey); err != nil {
		return fmt.Errorf("failed to delete from p2p: %w", err)
	}

	// Limpar cache
	cacheKey := fmt.Sprintf("p2p:file:%s", fileKey)
	userFilesKey := fmt.Sprintf("p2p:user:%d:files", userID)

	pipe := pa.redisCache.Pipeline()
	pipe.Del(ctx, cacheKey)
	pipe.SRem(ctx, userFilesKey, fileKey)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}

	return nil
}
