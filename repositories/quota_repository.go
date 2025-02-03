package repositories

import (
	"SafeBox/models"
	"context"
	"gorm.io/gorm"
)

type QuotaRepositoryInterface interface {
	GetUserQuota(ctx context.Context, userID uint) (*models.UserQuota, error)
	UpdateUserQuota(ctx context.Context, quota *models.UserQuota) error
	GetTotalUsage(ctx context.Context, userID uint) (int64, error)
	ReconcileUserQuota(ctx context.Context, userID uint) error // Adicionando este método
}

type QuotaRepository struct {
	db *gorm.DB
}

func NewQuotaRepository(db *gorm.DB) QuotaRepositoryInterface {
	return &QuotaRepository{db: db}
}

func (qr *QuotaRepository) GetUserQuota(ctx context.Context, userID uint) (*models.UserQuota, error) {
	var quota models.UserQuota
	if err := qr.db.Where("user_id = ?", userID).First(&quota).Error; err != nil {
		return nil, err
	}
	return &quota, nil
}

func (qr *QuotaRepository) UpdateUserQuota(ctx context.Context, quota *models.UserQuota) error {
	return qr.db.Save(quota).Error
}

func (qr *QuotaRepository) GetTotalUsage(ctx context.Context, userID uint) (int64, error) {
	var total int64
	if err := qr.db.Model(&models.UserQuota{}).Where("user_id = ?", userID).Sum("size", &total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (qr *QuotaRepository) ReconcileUserQuota(ctx context.Context, userID uint) error {
	actualUsed, err := qr.GetTotalUsage(ctx, userID)
	if err != nil {
		return err
	}
	quota, err := qr.GetUserQuota(ctx, userID)
	if err != nil {
		return err
	}
	quota.Used = actualUsed
	return qr.UpdateUserQuota(ctx, quota)
}
