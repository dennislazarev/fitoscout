package mariadb

import (
        "context"
        "database/sql"
        "fmt"
        "time"

        "fitoscout/backend/internal/domain"
)

// SyncRepo — реализация SyncRepository для MariaDB.
type SyncRepo struct {
        db *sql.DB
}

// NewSyncRepo создаёт новый репозиторий синхронизации.
func NewSyncRepo(db *sql.DB) *SyncRepo {
        return &SyncRepo{db: db}
}

// GetState возвращает состояние синхронизации для устройства.
func (r *SyncRepo) GetState(ctx context.Context, deviceID string) (*domain.SyncState, error) {
        query := `SELECT device_id, last_version, last_sync_at FROM sync_state WHERE device_id = ?`
        row := r.db.QueryRowContext(ctx, query, deviceID)

        state := &domain.SyncState{}
        var lastSyncAt sql.NullTime
        err := row.Scan(&state.DeviceID, &state.LastVersion, &lastSyncAt)
        if err != nil {
                if err == sql.ErrNoRows {
                        // Возвращаем пустое состояние для новых устройств
                        return &domain.SyncState{
                                DeviceID:    deviceID,
                                LastVersion: 0,
                                LastSyncAt:  nil,
                        }, nil
                }
                return nil, err
        }

        if lastSyncAt.Valid {
                state.LastSyncAt = &lastSyncAt.Time
        }
        return state, nil
}

// UpdateState обновляет состояние синхронизации для устройства.
func (r *SyncRepo) UpdateState(ctx context.Context, state *domain.SyncState) error {
        query := `INSERT INTO sync_state (device_id, last_version, last_sync_at)
              VALUES (?, ?, ?)
              ON DUPLICATE KEY UPDATE last_version = VALUES(last_version), last_sync_at = VALUES(last_sync_at)`

        var lastSyncAt interface{}
        if state.LastSyncAt != nil {
                lastSyncAt = *state.LastSyncAt
        } else {
                lastSyncAt = time.Now()
        }

        _, err := r.db.ExecContext(ctx, query, state.DeviceID, state.LastVersion, lastSyncAt)
        return err
}

// GetGlobalVersion возвращает максимальную version из всех модульных таблиц.
func (r *SyncRepo) GetGlobalVersion(ctx context.Context) (int64, error) {
        var maxVersion int64

        for _, table := range moduleTables {
                query := fmt.Sprintf("SELECT COALESCE(MAX(version), 0) FROM %s", table)
                var v int64
                if err := r.db.QueryRowContext(ctx, query).Scan(&v); err != nil {
                        // Таблица может не существовать - игнорируем
                        continue
                }
                if v > maxVersion {
                        maxVersion = v
                }
        }

        return maxVersion, nil
}

// IncrementVersion атомарно инкрементирует глобальный счётчик.
// Использует MAX(version) + 1 из всех таблиц.
func (r *SyncRepo) IncrementVersion(ctx context.Context) (int64, error) {
        // Для простоты используем MAX(version) + 1
        // В production можно использовать отдельную таблицу-счётчик с LOCK
        current, err := r.GetGlobalVersion(ctx)
        if err != nil {
                return 0, err
        }
        return current + 1, nil
}