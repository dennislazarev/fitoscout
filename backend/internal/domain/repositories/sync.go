package repositories

import (
        "context"
        "fitoscout/backend/internal/domain"
)

// SyncRepository — репозиторий для состояния синхронизации.
type SyncRepository interface {
        GetState(ctx context.Context, deviceID string) (*domain.SyncState, error)
        UpdateState(ctx context.Context, state *domain.SyncState) error

        // GetGlobalVersion возвращает текущую глобальную версию (максимальная version во всех таблицах).
        GetGlobalVersion(ctx context.Context) (int64, error)

        // IncrementVersion атомарно инкрементирует глобальный счётчик версий.
        // Используется при создании/обновлении записей для LWW.
        IncrementVersion(ctx context.Context) (int64, error)
}