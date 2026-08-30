package repositories

import (
        "context"
        "fitoscout/backend/internal/domain"
)

// EAVRepository — репозиторий для EAV (характеристики).
type EAVRepository interface {
        // Определения
        GetDefinitionsByModule(ctx context.Context, moduleKey string) ([]domain.AttributeDefinition, error)
        CreateDefinition(ctx context.Context, def *domain.AttributeDefinition) error
        UpdateDefinition(ctx context.Context, def *domain.AttributeDefinition) error
        DeleteDefinition(ctx context.Context, id int64) error

        // Значения
        GetValuesForEntity(ctx context.Context, moduleKey, entityID string) ([]domain.AttributeValue, error)
        SetValue(ctx context.Context, value *domain.AttributeValue) error
        DeleteValuesForEntity(ctx context.Context, moduleKey, entityID string) error
}