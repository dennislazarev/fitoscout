package repositories

import (
	"context"
	"fitoscout/backend/internal/domain"
)

// DictionaryRepository — репозиторий для словарей.
type DictionaryRepository interface {
	GetAll(ctx context.Context) ([]domain.Dictionary, error)
	GetByID(ctx context.Context, id int64) (*domain.Dictionary, error)
	GetWithItems(ctx context.Context, id int64) (*domain.Dictionary, []domain.DictionaryItem, error)
	Create(ctx context.Context, dict *domain.Dictionary) error
	Update(ctx context.Context, dict *domain.Dictionary) error
	Delete(ctx context.Context, id int64) error
}
