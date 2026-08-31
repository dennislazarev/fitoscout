package repositories

import (
	"context"
	"fitoscout/backend/internal/domain"
)

// CategoryRepository — репозиторий для категорий.
type CategoryRepository interface {
	GetByModule(ctx context.Context, moduleKey string) ([]domain.Category, error)
	GetByID(ctx context.Context, id int64) (*domain.Category, error)
	GetTree(ctx context.Context, moduleKey string) ([]domain.Category, error) // иерархия
	Create(ctx context.Context, category *domain.Category) error
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id int64) error
}
