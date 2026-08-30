// Package repositories содержит интерфейсы репозиториев (ADR-001:
// Clean Architecture, dependency inversion).
package repositories

import (
        "context"
)

// Repository — generic CRUD-репозиторий для версионируемых сущностей.
// T — тип сущности (Plant, Disease, ...).
type Repository[T any] interface {
        // GetByID возвращает сущность по ID (только активные, deleted_at IS NULL).
        GetByID(ctx context.Context, id string) (*T, error)

        // List возвращает список сущностей с пагинацией (только активные).
        List(ctx context.Context, limit, offset int) ([]T, error)

        // Count возвращает общее количество активных записей.
        Count(ctx context.Context) (int64, error)

        // Create создаёт новую сущность с UUIDv7, version=1, timestamps.
        Create(ctx context.Context, entity *T) error

        // Update обновляет сущность с проверкой версии (LWW).
        // Если текущая version в БД != entity.Version → возвращает ErrConflict.
        // При успехе инкрементирует version и обновляет updated_at.
        Update(ctx context.Context, entity *T) error

        // SoftDelete устанавливает deleted_at (мягкое удаление).
        SoftDelete(ctx context.Context, id string) error

        // GetSince возвращает все записи с version > since (для дельта-синхронизации).
        // Включает soft-deleted записи (tombstones).
        GetSince(ctx context.Context, since int64) ([]T, error)
}