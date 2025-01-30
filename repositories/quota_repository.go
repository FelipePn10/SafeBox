// Package repositories gerencia o acesso aos dados de cota
package repositories

import (
	"SafeBox/models"
	"context"
	"gorm.io/gorm"
)

type QuotaRepositoryInterface interface {
	GetUserQuota(ctx context.Context, userID uint) (*models.UserQuota, error)
	UpdateUserQuota(ctx context.Context, userQuota *models.UserQuota) error
	CreateInitialQuota(ctx context.Context, userID uint) error
	GetAllUsers(ctx context.Context) ([]models.UserQuota, error)
	UpdateUsage(ctx context.Context, userID uint, used int64) error
}

type QuotaRepository struct {
	db *gorm.DB
}

func NewQuotaRepository(db *gorm.DB) QuotaRepositoryInterface {
	return &QuotaRepository{db: db}
}

func (r *QuotaRepository) GetUserQuota(ctx context.Context, userID uint) (*models.UserQuota, error) {
	var quota models.UserQuota
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&quota)
	return &quota, result.Error
}

func (r *QuotaRepository) UpdateUserQuota(ctx context.Context, userQuota *models.UserQuota) error {
	return r.db.WithContext(ctx).Save(userQuota).Error
}

func (r *QuotaRepository) CreateInitialQuota(ctx context.Context, userID uint) error {
	quota := &models.UserQuota{
		UserID: userID,
		Plan:   models.Free,
	}
	quota.SetDefaults()
	return r.db.WithContext(ctx).Create(quota).Error
}

func (r *QuotaRepository) GetAllUsers(ctx context.Context) ([]models.UserQuota, error) {
	var quotas []models.UserQuota
	result := r.db.WithContext(ctx).Find(&quotas)
	return quotas, result.Error
}

func (r *QuotaRepository) UpdateUsage(ctx context.Context, userID uint, used int64) error {
	return r.db.WithContext(ctx).Model(&models.UserQuota{}).
		Where("user_id = ?", userID).
		Update("used", used).Error
}
