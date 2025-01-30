package graph

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"gorm.io/gorm"
	"net/http"
)

// Resolver serves as dependency injection for your app, add any dependencies you require here.
type Resolver struct {
	DB *gorm.DB
}

func NewExecutableSchema(cfg Config) graphql.ExecutableSchema {
	return cfg
}

type Config struct {
	Resolvers ResolverRoot
}

func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Query() QueryResolver       { return &queryResolver{r} }
func (r *Resolver) User() UserResolver         { return &userResolver{r} }

func NewGraphQLHandler(db *gorm.DB) *handler.Server {
	return handler.NewDefaultServer(NewExecutableSchema(Config{
		Resolvers: &Resolver{DB: db},
	}))
}

func PlaygroundHandler() http.HandlerFunc {
	return playground.Handler("GraphQL", "/query")
}
