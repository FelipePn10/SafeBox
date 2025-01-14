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

// Create insere um novo usuário no banco de dados
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// FindByUsername busca um usuário pelo nome de usuário
func (r UserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	result := r.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// Update atualiza um usuário no banco de dados
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}
