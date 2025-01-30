package services

import (
	"SafeBox/models"
	"SafeBox/repositories"
	"SafeBox/services/storage"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type QuotaServiceInterface interface {
	GetCurrentUsage(ctx context.Context, userID uint) (int64, error)
	CheckAndReserveSpace(ctx context.Context, userID uint, fileSize int64) error
	CommitSpaceUsage(ctx context.Context, userID uint, fileSize int64) error
	RollbackSpaceReservation(ctx context.Context, userID uint, fileSize int64)
	GetLimit(userID uint) int64
}

type QuotaService struct {
	repo        repositories.QuotaRepositoryInterface
	storageRepo storage.StorageRepository
	redisClient *redis.Client
}

func NewQuotaService(
	repo repositories.QuotaRepositoryInterface,
	storageRepo storage.StorageRepository,
	redisClient *redis.Client,
) *QuotaService {
	return &QuotaService{
		repo:        repo,
		storageRepo: storageRepo,
		redisClient: redisClient,
	}
}

func (qs *QuotaService) GetCurrentUsage(ctx context.Context, userID uint) (int64, error) {
	cacheKey := fmt.Sprintf("quota:%d:used", userID)

	if val, err := qs.redisClient.Get(ctx, cacheKey).Int64(); err == nil {
		return val, nil
	}

	totalUsed, err := qs.storageRepo.GetTotalUsage(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("storage calculation failed: %w", err)
	}

	qs.redisClient.Set(ctx, cacheKey, totalUsed, 5*time.Minute)
	return totalUsed, nil
}

func (qs *QuotaService) CheckAndReserveSpace(ctx context.Context, userID uint, fileSize int64) error {
	quota, err := qs.repo.GetUserQuota(ctx, userID)
	if err != nil {
		return fmt.Errorf("quota lookup failed: %w", err)
	}

	currentUsed, err := qs.GetCurrentUsage(ctx, userID)
	if err != nil {
		return err
	}

	if currentUsed+fileSize > quota.Limit {
		return models.ErrStorageLimitExceeded
	}

	tempKey := fmt.Sprintf("quota:%d:temp", userID)
	return qs.redisClient.IncrBy(ctx, tempKey, fileSize).Err()
}

func (qs *QuotaService) CommitSpaceUsage(ctx context.Context, userID uint, fileSize int64) error {
	quota, err := qs.repo.GetUserQuota(ctx, userID)
	if err != nil {
		return err
	}

	quota.Used += fileSize
	if err := qs.repo.UpdateUserQuota(ctx, quota); err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	cacheKey := fmt.Sprintf("quota:%d:used", userID)
	qs.redisClient.IncrBy(ctx, cacheKey, fileSize)

	tempKey := fmt.Sprintf("quota:%d:temp", userID)
	qs.redisClient.Del(ctx, tempKey)

	return nil
}

func (qs *QuotaService) RollbackSpaceReservation(ctx context.Context, userID uint, fileSize int64) {
	tempKey := fmt.Sprintf("quota:%d:temp", userID)
	qs.redisClient.DecrBy(ctx, tempKey, fileSize)
}

func (qs *QuotaService) GetLimit(userID uint) int64 {
	quota, _ := qs.repo.GetUserQuota(context.Background(), userID)
	return quota.Limit
}
