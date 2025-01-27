package graph

import (
	"SafeBox/graph/model"
	"SafeBox/models"
	"context"
	"fmt"
	"gorm.io/gorm"
)

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type userResolver struct{ *Resolver }

// CreateUser is the resolver for the createUser field.
func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUserInput) (*models.OAuthUser, error) {
	user := &models.OAuthUser{
		Username: input.Username,
		Email:    input.Email,
		// Outros campos necessários
	}
	if err := r.Resolver.DB.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser is the resolver for the getUser field.
func (r *queryResolver) GetUser(ctx context.Context, id string) (*models.OAuthUser, error) {
	var user models.OAuthUser
	if err := r.Resolver.DB.Where("username = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// ListUsers is the resolver for the listUsers field.
func (r *queryResolver) ListUsers(ctx context.Context) ([]*models.OAuthUser, error) {
	var users []*models.OAuthUser
	if err := r.Resolver.DB.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ID is the resolver for the id field.
func (r *userResolver) ID(ctx context.Context, obj *models.OAuthUser) (string, error) {
	return obj.Username, nil
}
