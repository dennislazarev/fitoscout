package mariadb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
	apperrors "fitoscout/backend/internal/errors"
)

// moduleTables — список модульных таблиц для синхронизации.
var moduleTables = []string{
	"plants", "diseases", "pests", "agrochemicals",
	"active_substances", "terminologies", "articles",
	"registry", "calendar", "library", "comments",
}

// ModuleRepositoryMap — карта репозиториев по ключам модулей.
type ModuleRepositoryMap map[string]repositories.Repository[any]

// NewModuleRepositoryMap создаёт фабрику репозиториев для всех 11 модулей.
func NewModuleRepositoryMap(db *sql.DB, syncRepo repositories.SyncRepository) ModuleRepositoryMap {
	repos := make(ModuleRepositoryMap)

	for _, table := range moduleTables {
		repo := NewBaseRepository[any](db, table, syncRepo)
		repos[table] = repo
	}

	return repos
}

// GetDeltaForModules возвращает изменения из всех модулей с version > since.
func (m ModuleRepositoryMap) GetDeltaForModules(ctx context.Context, since int64) (map[string][]any, int64, error) {
	result := make(map[string][]any)
	maxVersion := since

	for moduleKey, repo := range m {
		entities, err := repo.GetSince(ctx, since)
		if err != nil {
			// Игнорируем ошибки для отдельных таблиц (могут не существовать)
			continue
		}

		if len(entities) > 0 {
			// Преобразуем entities в slice of any
			data := make([]any, len(entities))
			for i, e := range entities {
				data[i] = e
			}
			result[moduleKey] = data

			// Обновляем maxVersion
			for _, e := range entities {
				if v, ok := getVersion(e); ok && v > maxVersion {
					maxVersion = v
				}
			}
		}
	}

	return result, maxVersion, nil
}

// ApplyChange применяет одно изменение от клиента с LWW-логикой.
func (m ModuleRepositoryMap) ApplyChange(ctx context.Context, change ClientChange) (int64, error) {
	repo, exists := m[change.Module]
	if !exists {
		return 0, apperrors.UnknownModule(fmt.Sprintf("неизвестный модуль: %s", change.Module))
	}

	switch change.Action {
	case "create":
		return m.applyCreate(ctx, repo, change)
	case "update":
		return m.applyUpdate(ctx, repo, change)
	case "delete":
		return m.applyDelete(ctx, repo, change)
	default:
		return 0, apperrors.BadRequest(fmt.Sprintf("неизвестное действие: %s", change.Action))
	}
}

// applyCreate создаёт новую запись.
func (m ModuleRepositoryMap) applyCreate(ctx context.Context, repo repositories.Repository[any], change ClientChange) (int64, error) {
	entity, err := createEntityFromData(change.Module, change.Data)
	if err != nil {
		return 0, err
	}

	if err := repo.Create(ctx, entity); err != nil {
		return 0, err
	}

	// Получаем созданную сущность для возврата версии
	id := getID(entity)
	if id == "" {
		return 0, apperrors.Internal("не удалось получить ID созданной записи")
	}

	created, err := repo.GetByID(ctx, id)
	if err != nil {
		return 0, err
	}

	v, _ := getVersion(created)
	return v, nil
}

// applyUpdate обновляет запись с LWW-проверкой.
func (m ModuleRepositoryMap) applyUpdate(ctx context.Context, repo repositories.Repository[any], change ClientChange) (int64, error) {
	// Получаем текущую версию из БД
	current, err := repo.GetByID(ctx, change.ID)
	if err != nil {
		if isNotFound(err) {
			return 0, apperrors.NotFound("запись не найдена")
		}
		return 0, err
	}

	serverVersion, _ := getVersion(current)

	// LWW: если версия сервера > версии клиента → конфликт
	if serverVersion > change.Version {
		return serverVersion, apperrors.ConflictWithVersion(serverVersion)
	}

	// Применяем обновление
	entity, err := updateEntityFromData(change.Module, change.Data, current)
	if err != nil {
		return 0, err
	}

	if err := repo.Update(ctx, entity); err != nil {
		return 0, err
	}

	// Возвращаем новую версию
	updated, err := repo.GetByID(ctx, change.ID)
	if err != nil {
		return 0, err
	}

	v, _ := getVersion(updated)
	return v, nil
}

// applyDelete выполняет soft delete.
func (m ModuleRepositoryMap) applyDelete(ctx context.Context, repo repositories.Repository[any], change ClientChange) (int64, error) {
	// Проверяем существование записи
	_, err := repo.GetByID(ctx, change.ID)
	if err != nil {
		if isNotFound(err) {
			return 0, apperrors.NotFound("запись не найдена")
		}
		return 0, err
	}

	if err := repo.SoftDelete(ctx, change.ID); err != nil {
		return 0, err
	}

	// Получаем удалённую запись для возврата версии
	deleted, err := repo.GetByID(ctx, change.ID)
	if err != nil {
		// После soft delete запись может быть не найдена через GetByID
		// Это нормально, возвращаем версию из change + инкремент
		return change.Version + 1, nil
	}

	v, _ := getVersion(deleted)
	return v, nil
}

// Helper functions

func getVersion(entity any) (int64, bool) {
	if v, ok := entity.(interface{ GetVersion() int64 }); ok {
		return v.GetVersion(), true
	}
	return 0, false
}

func getID(entity any) string {
	if v, ok := entity.(interface{ GetID() string }); ok {
		return v.GetID()
	}
	return ""
}

func isNotFound(err error) bool {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == apperrors.CodeNotFound
	}
	return false
}

// createEntityFromData создаёт сущность из JSON данных.
func createEntityFromData(module string, data any) (*any, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var entity any
	switch module {
	case "plants":
		entity = &domain.Plant{}
	case "diseases":
		entity = &domain.Disease{}
	case "pests":
		entity = &domain.Pest{}
	case "agrochemicals":
		entity = &domain.Agrochemical{}
	case "active_substances":
		entity = &domain.ActiveSubstance{}
	case "terminologies":
		entity = &domain.Terminology{}
	case "articles":
		entity = &domain.Article{}
	case "registry":
		entity = &domain.RegistryItem{}
	case "calendar":
		entity = &domain.CalendarEvent{}
	case "library":
		entity = &domain.LibraryItem{}
	case "comments":
		entity = &domain.Comment{}
	default:
		return nil, apperrors.UnknownModule(fmt.Sprintf("неизвестный модуль: %s", module))
	}

	if err := json.Unmarshal(jsonData, entity); err != nil {
		return nil, err
	}

	return &entity, nil
}

// updateEntityFromData обновляет сущность из JSON данных.
func updateEntityFromData(module string, data any, existing any) (*any, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(jsonData, existing); err != nil {
		return nil, err
	}

	return &existing, nil
}

// ClientChange — изменение от клиента (дублируем для удобства).
type ClientChange struct {
	Module  string `json:"module"`
	ID      string `json:"id"`
	Action  string `json:"action"` // "create", "update", "delete"
	Data    any    `json:"data"`
	Version int64  `json:"version"`
}
