package repositories

import (
	"SafeBox/models"

	"gorm.io/gorm"
)

type TwoFactorRepository struct {
	db *gorm.DB
}

func NewTwoFactorRepository(db *gorm.DB) *TwoFactorRepository {
	return &TwoFactorRepository{db: db}
}

func (r *TwoFactorRepository) SaveTempSecret(secret *models.TempTwoFASecret) error {
	return r.db.Create(secret).Error
}

func (r *TwoFactorRepository) FindTempSecretByEmail(email string) (*models.TempTwoFASecret, error) {
	var secret models.TempTwoFASecret
	err := r.db.Where("user_email = ?", email).First(&secret).Error
	return &secret, err
}

func (r *TwoFactorRepository) DeleteTempSecret(secret *models.TempTwoFASecret) error {
	return r.db.Delete(secret).Error
}
