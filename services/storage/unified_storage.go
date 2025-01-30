package storage

import (
	"context"
	"fmt"
	"io"
)

type StorageType int

const (
	Local StorageType = iota
	P2P
	R2
)

type StorageRepository interface {
	Save(ctx context.Context, file io.Reader, userID uint, fileName string) error
	GetTotalUsage(ctx context.Context, userID uint) (int64, error)
	Delete(ctx context.Context, userID uint, fileName string) error
}

type UnifiedStorage struct {
	local StorageRepository
	p2p   StorageRepository
	r2    StorageRepository
}

func NewUnifiedStorage(local StorageRepository, p2p StorageRepository, r2 StorageRepository) *UnifiedStorage {
	return &UnifiedStorage{
		local: local,
		p2p:   p2p,
		r2:    r2,
	}
}

func (us *UnifiedStorage) Save(ctx context.Context, file io.Reader, userID uint, fileName string) error {
	// Implementar lógica de escolha do storage baseado em regras de negócio
	// Arquivos grandes vão para R2, pequenos para local.

	// Por enquanto, salvando em todos os storages disponíveis
	if us.local != nil {
		if err := us.local.Save(ctx, file, userID, fileName); err != nil {
			return fmt.Errorf("local storage error: %w", err)
		}
	}

	if us.p2p != nil {
		if err := us.p2p.Save(ctx, file, userID, fileName); err != nil {
			return fmt.Errorf("p2p storage error: %w", err)
		}
	}

	if us.r2 != nil {
		if err := us.r2.Save(ctx, file, userID, fileName); err != nil {
			return fmt.Errorf("r2 storage error: %w", err)
		}
	}

	return nil
}

func (us *UnifiedStorage) GetTotalUsage(ctx context.Context, userID uint) (int64, error) {
	var total int64
	var errs []error

	if us.local != nil {
		if usage, err := us.local.GetTotalUsage(ctx, userID); err != nil {
			errs = append(errs, fmt.Errorf("local storage error: %w", err))
		} else {
			total += usage
		}
	}

	if us.p2p != nil {
		if usage, err := us.p2p.GetTotalUsage(ctx, userID); err != nil {
			errs = append(errs, fmt.Errorf("p2p storage error: %w", err))
		} else {
			total += usage
		}
	}

	if us.r2 != nil {
		if usage, err := us.r2.GetTotalUsage(ctx, userID); err != nil {
			errs = append(errs, fmt.Errorf("r2 storage error: %w", err))
		} else {
			total += usage
		}
	}

	if len(errs) > 0 {
		return total, fmt.Errorf("storage errors: %v", errs)
	}

	return total, nil
}

func (us *UnifiedStorage) Delete(ctx context.Context, userID uint, fileName string) error {
	var errs []error

	if us.local != nil {
		if err := us.local.Delete(ctx, userID, fileName); err != nil {
			errs = append(errs, fmt.Errorf("local storage error: %w", err))
		}
	}

	if us.p2p != nil {
		if err := us.p2p.Delete(ctx, userID, fileName); err != nil {
			errs = append(errs, fmt.Errorf("p2p storage error: %w", err))
		}
	}

	if us.r2 != nil {
		if err := us.r2.Delete(ctx, userID, fileName); err != nil {
			errs = append(errs, fmt.Errorf("r2 storage error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("delete errors: %v", errs)
	}

	return nil
}
