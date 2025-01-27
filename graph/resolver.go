package graph

import (
	"gorm.io/gorm"
)

// Resolver serves as dependency injection for your app, add any dependencies you require here.
type Resolver struct {
	DB *gorm.DB
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() QueryResolver       { return &queryResolver{r} }
func (r *Resolver) User() UserResolver         { return &userResolver{r} }
