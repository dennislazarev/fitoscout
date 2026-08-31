package repositories

import (
	"context"
	"fitoscout/backend/internal/domain"
)

// ModuleRepository — репозиторий для реестра модулей.
type ModuleRepository interface {
	GetAll(ctx context.Context) ([]domain.Module, error)
	GetByID(ctx context.Context, id string) (*domain.Module, error)
	GetActive(ctx context.Context) ([]domain.Module, error)
}
