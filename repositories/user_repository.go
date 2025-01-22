// repositories/user_repository.go
package repositories

import (
	"SafeBox/models"
	"fmt"

	"gorm.io/gorm"
)

// UserRepository defines the interface for user operations
type UserRepository interface {
	CreateOrUpdate(user *models.OAuthUser) error
	FindByUsername(username string) (*models.OAuthUser, error)
	FindByEmail(email string) (*models.OAuthUser, error)
	UpdateStorageUsed(username string, size int64) error
	CreateUser(user *models.OAuthUser) error
	Update(user *models.OAuthUser) error
}

// userRepositoryImpl implements UserRepository interface
type userRepositoryImpl struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryImpl{
		db: db,
	}
}

func (r *userRepositoryImpl) CreateUser(user *models.OAuthUser) error {
	return r.db.Create(user).Error
}

func (r *userRepositoryImpl) FindByUsername(username string) (*models.OAuthUser, error) {
	var user models.OAuthUser
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepositoryImpl) FindByEmail(email string) (*models.OAuthUser, error) {
	var user models.OAuthUser
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

func (r *userRepositoryImpl) Update(user *models.OAuthUser) error {
	return r.db.Save(user).Error
}

func (r *userRepositoryImpl) UpdateStorageUsed(username string, size int64) error {
	return r.db.Model(&models.OAuthUser{}).
		Where("username = ?", username).
		UpdateColumn("storage_used", gorm.Expr("storage_used + ?", size)).
		Error
}

func (r *userRepositoryImpl) CreateOrUpdate(user *models.OAuthUser) error {
	return r.db.Where("oauth_id = ?", user.OAuthID).
		Assign(models.OAuthUser{
			Username:     user.Username,
			Avatar:       user.Avatar,
			AccessToken:  user.AccessToken,
			RefreshToken: user.RefreshToken,
			TokenExpiry:  user.TokenExpiry,
		}).
		FirstOrCreate(user).Error
}
