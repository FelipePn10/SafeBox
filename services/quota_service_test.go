package services

import (
	"SafeBox/repositories"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateUsedSpace(t *testing.T) {
	repo := repositories.NewMockQuotaRepository()
	service := NewQuotaService(repo, nil)

	used, err := service.CalculateUsedSpace(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), used)
}

func TestUpdateUsedSpace(t *testing.T) {
	repo := repositories.NewMockQuotaRepository()
	service := NewQuotaService(repo, nil)

	err := service.UpdateUsedSpace(context.Background(), 1, 50)
	assert.NoError(t, err)
}
