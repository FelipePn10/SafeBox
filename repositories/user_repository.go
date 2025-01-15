package repositories

import (
	"SafeBox/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *models.OAuthUser) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByUsername(username string) (*models.OAuthUser, error) {
	var user models.OAuthUser
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

// Update atualiza um usuário no banco de dados
func (r *UserRepository) Update(user *models.OAuthUser) error {
	return r.db.Save(user).Error
}
