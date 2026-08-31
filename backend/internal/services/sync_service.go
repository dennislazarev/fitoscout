// Package services содержит реализацию сервиса синхронизации.
package services

import (
	"context"
	"errors"
	"time"

	"fitoscout/backend/internal/domain/repositories"
	"fitoscout/backend/internal/domain/services"
	apperrors "fitoscout/backend/internal/errors"
	"fitoscout/backend/internal/storage/mariadb"
)

// syncServiceImpl — реализация SyncService.
type syncServiceImpl struct {
	syncRepo    repositories.SyncRepository
	moduleRepos mariadb.ModuleRepositoryMap
}

// NewSyncService создаёт новый сервис синхронизации.
func NewSyncService(syncRepo repositories.SyncRepository, moduleRepos mariadb.ModuleRepositoryMap) services.SyncService {
	return &syncServiceImpl{
		syncRepo:    syncRepo,
		moduleRepos: moduleRepos,
	}
}

// GetDelta возвращает дельту изменений с версии since.
func (s *syncServiceImpl) GetDelta(ctx context.Context, deviceID string, since int64) (*services.SyncDelta, error) {
	// Получаем изменения из всех модулей
	changes, maxVersion, err := s.moduleRepos.GetDeltaForModules(ctx, since)
	if err != nil {
		return nil, err
	}

	// Обновляем состояние устройства
	state, err := s.syncRepo.GetState(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if maxVersion > state.LastVersion {
		state.LastVersion = maxVersion
		now := time.Now()
		state.LastSyncAt = &now
		if err := s.syncRepo.UpdateState(ctx, state); err != nil {
			return nil, err
		}
	}

	return &services.SyncDelta{
		DeviceID:       deviceID,
		SinceVersion:   since,
		CurrentVersion: maxVersion,
		Changes:        changes,
		ServerTime:     time.Now().UnixMilli(),
	}, nil
}

// ApplyChanges применяет изменения от клиента с LWW-логикой.
func (s *syncServiceImpl) ApplyChanges(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
	result := &services.ApplyResult{
		Accepted: make([]services.AppliedChange, 0),
		Rejected: make([]services.RejectedChange, 0),
	}

	// Применяем каждое изменение
	for _, change := range changes {
		newVersion, err := s.moduleRepos.ApplyChange(ctx, mariadb.ClientChange{
			Module:  change.Module,
			ID:      change.ID,
			Action:  change.Action,
			Data:    change.Data,
			Version: change.Version,
		})

		if err != nil {
			// Определяем причину отказа
			reason := s.determineRejectReason(err)
			serverVersion := int64(0)

			// Извлекаем версию сервера из ошибки конфликта
			var appErr *apperrors.AppError
			if errors.As(err, &appErr) && appErr.Code == apperrors.CodeConflict {
				if details, ok := appErr.Details.(map[string]int64); ok {
					if sv, exists := details["server_version"]; exists {
						serverVersion = sv
					}
				}
			}

			result.Rejected = append(result.Rejected, services.RejectedChange{
				Module:        change.Module,
				ID:            change.ID,
				Reason:        reason,
				ServerVersion: serverVersion,
			})
		} else {
			result.Accepted = append(result.Accepted, services.AppliedChange{
				Module:     change.Module,
				ID:         change.ID,
				NewVersion: newVersion,
			})
		}
	}

	// После применения всех изменений получаем дельту сервера
	currentVersion, _ := s.syncRepo.GetGlobalVersion(ctx)
	delta := &services.SyncDelta{
		DeviceID:       deviceID,
		SinceVersion:   currentVersion,
		CurrentVersion: currentVersion,
		Changes:        make(map[string][]any),
		ServerTime:     time.Now().UnixMilli(),
	}

	// Если были принятые изменения, обновляем состояние устройства
	if len(result.Accepted) > 0 {
		state, err := s.syncRepo.GetState(ctx, deviceID)
		if err == nil {
			now := time.Now()
			state.LastSyncAt = &now
			if currentVersion > state.LastVersion {
				state.LastVersion = currentVersion
			}
			_ = s.syncRepo.UpdateState(ctx, state)
		}
	}

	result.Delta = delta
	return result, nil
}

// determineRejectReason определяет причину отказа на основе ошибки.
func (s *syncServiceImpl) determineRejectReason(err error) string {
	if err == nil {
		return "unknown"
	}

	errStr := err.Error()

	// Проверяем на конфликт версий
	if contains(errStr, "конфликт версий") || contains(errStr, "conflict") {
		return "conflict_higher_version"
	}

	// Проверяем на не найденную запись
	if contains(errStr, "не найдена") || contains(errStr, "not found") || contains(errStr, "not_found") {
		return "not_found"
	}

	// Проверяем на неизвестный модуль
	if contains(errStr, "неизвестный модуль") || contains(errStr, "unknown_module") {
		return "unknown_module"
	}

	return "unknown"
}

// contains проверяет наличие подстроки в строке.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
