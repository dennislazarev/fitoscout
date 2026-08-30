// Package mariadb содержит реализации репозиториев для MariaDB.
package mariadb

import (
        "context"
        "database/sql"
        "fmt"
        "reflect"
        "strings"
        "time"

        "github.com/google/uuid"

        apperrors "fitoscout/backend/internal/errors"
        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// BaseRepository — generic реализация Repository[T] для MariaDB.
type BaseRepository[T any] struct {
        db         *sql.DB
        tableName  string
        syncRepo   repositories.SyncRepository
}

// NewBaseRepository создаёт новый базовый репозиторий.
func NewBaseRepository[T any](db *sql.DB, tableName string, syncRepo repositories.SyncRepository) *BaseRepository[T] {
        return &BaseRepository[T]{
                db:        db,
                tableName: tableName,
                syncRepo:  syncRepo,
        }
}

// GetByID возвращает сущность по ID (только активные).
func (r *BaseRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
        query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? AND deleted_at IS NULL", r.tableName)
        row := r.db.QueryRowContext(ctx, query, id)

        entity := new(T)
        if err := scanRow(row, entity); err != nil {
                if err == sql.ErrNoRows {
                        return nil, apperrors.NotFound("запись не найдена")
                }
                return nil, err
        }
        return entity, nil
}

// List возвращает список сущностей с пагинацией (только активные).
func (r *BaseRepository[T]) List(ctx context.Context, limit, offset int) ([]T, error) {
        query := fmt.Sprintf("SELECT * FROM %s WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?", r.tableName)
        rows, err := r.db.QueryContext(ctx, query, limit, offset)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var entities []T
        for rows.Next() {
                var entity T
                if err := scanRowTo(rows, &entity); err != nil {
                        return nil, err
                }
                entities = append(entities, entity)
        }
        return entities, rows.Err()
}

// Count возвращает общее количество активных записей.
func (r *BaseRepository[T]) Count(ctx context.Context) (int64, error) {
        query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted_at IS NULL", r.tableName)
        var count int64
        err := r.db.QueryRowContext(ctx, query).Scan(&count)
        return count, err
}

// Create создаёт новую сущность с UUIDv7, version=1, timestamps.
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
        // Генерация UUIDv7
        id := uuid.New().String()

        // Установка базовых полей через интерфейсы
        if identifiable, ok := any(entity).(domain.Identifiable); ok {
                identifiable.SetID(id)
        }

        // Получение новой версии из глобального счётчика
        newVersion, err := r.syncRepo.IncrementVersion(ctx)
        if err != nil {
                return err
        }

        if versionable, ok := any(entity).(domain.Versionable); ok {
                versionable.SetVersion(newVersion)
        }

        now := time.Now()
        if timestampable, ok := any(entity).(domain.Timestampable); ok {
                timestampable.SetCreatedAt(now)
                timestampable.SetUpdatedAt(now)
        }

        // Генерация SQL INSERT через reflection
        columns, values := buildInsertColumns(entity)
        placeholders := make([]string, len(columns))
        for i := range placeholders {
                placeholders[i] = "?"
        }

        query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", r.tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

        result, err := r.db.ExecContext(ctx, query, values...)
        if err != nil {
                return fmt.Errorf("ошибка вставки: %w", err)
        }

        // Для таблиц без AUTO_INCREMENT (id - UUID) просто возвращаем nil
        _ = result
        return nil
}

// Update обновляет сущность с проверкой версии (LWW).
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
        // Получение текущей версии из БД
        var currentVersion int64
        query := fmt.Sprintf("SELECT version FROM %s WHERE id = ? AND deleted_at IS NULL", r.tableName)

        id := ""
        if identifiable, ok := any(entity).(domain.Identifiable); ok {
                id = identifiable.GetID()
        }

        err := r.db.QueryRowContext(ctx, query, id).Scan(&currentVersion)
        if err != nil {
                if err == sql.ErrNoRows {
                        return apperrors.NotFound("запись не найдена")
                }
                return err
        }

        // Проверка версии (LWW)
        var entityVersion int64
        if versionable, ok := any(entity).(domain.Versionable); ok {
                entityVersion = versionable.GetVersion()
                if entityVersion != currentVersion {
                        return apperrors.Conflict("запись была изменена другим пользователем")
                }

                // Инкремент версии
                newVersion, err := r.syncRepo.IncrementVersion(ctx)
                if err != nil {
                        return err
                }
                versionable.SetVersion(newVersion)
        }

        // Обновление updated_at
        if timestampable, ok := any(entity).(domain.Timestampable); ok {
                timestampable.SetUpdatedAt(time.Now())
        }

        // Генерация SQL UPDATE через reflection
        setClause, values := buildUpdateSet(entity)
        values = append(values, id)

        query = fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", r.tableName, setClause)
        _, err = r.db.ExecContext(ctx, query, values...)
        return err
}

// SoftDelete устанавливает deleted_at (мягкое удаление).
func (r *BaseRepository[T]) SoftDelete(ctx context.Context, id string) error {
        now := time.Now()
        query := fmt.Sprintf("UPDATE %s SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL", r.tableName)

        // Инкремент версии при удалении
        newVersion, err := r.syncRepo.IncrementVersion(ctx)
        if err != nil {
                return err
        }

        // Добавляем обновление версии
        query = fmt.Sprintf("UPDATE %s SET deleted_at = ?, updated_at = ?, version = ? WHERE id = ? AND deleted_at IS NULL", r.tableName)

        _, err = r.db.ExecContext(ctx, query, now, now, newVersion, id)
        return err
}

// GetSince возвращает все записи с version > since (для дельта-синхронизации).
// Включает soft-deleted записи (tombstones).
func (r *BaseRepository[T]) GetSince(ctx context.Context, since int64) ([]T, error) {
        query := fmt.Sprintf("SELECT * FROM %s WHERE version > ? ORDER BY version ASC", r.tableName)
        rows, err := r.db.QueryContext(ctx, query, since)
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var entities []T
        for rows.Next() {
                var entity T
                if err := scanRowTo(rows, &entity); err != nil {
                        return nil, err
                }
                entities = append(entities, entity)
        }
        return entities, rows.Err()
}

// buildInsertColumns генерирует список колонок и значений для INSERT через reflection.
func buildInsertColumns(entity any) ([]string, []any) {
        v := reflect.ValueOf(entity).Elem()
        t := v.Type()

        var columns []string
        var values []any

        for i := 0; i < t.NumField(); i++ {
                field := t.Field(i)
                dbTag := field.Tag.Get("db")
                jsonTag := field.Tag.Get("json")

                if dbTag == "" || dbTag == "-" {
                        continue
                }

                // Пропускаем поля с omitempty, если они пустые
                if strings.Contains(jsonTag, "omitempty") {
                        val := v.Field(i)
                        if val.IsZero() {
                                continue
                        }
                }

                columns = append(columns, dbTag)
                values = append(values, v.Field(i).Interface())
        }

        return columns, values
}

// buildUpdateSet генерирует SET clause для UPDATE через reflection.
func buildUpdateSet(entity any) (string, []any) {
        v := reflect.ValueOf(entity).Elem()
        t := v.Type()

        var setParts []string
        var values []any

        for i := 0; i < t.NumField(); i++ {
                field := t.Field(i)
                dbTag := field.Tag.Get("db")
                jsonTag := field.Tag.Get("json")

                if dbTag == "" || dbTag == "-" || dbTag == "id" {
                        continue
                }

                // Пропускаем поля с omitempty, если они пустые
                if strings.Contains(jsonTag, "omitempty") {
                        val := v.Field(i)
                        if val.IsZero() {
                                continue
                        }
                }

                setParts = append(setParts, fmt.Sprintf("%s = ?", dbTag))
                values = append(values, v.Field(i).Interface())
        }

        return strings.Join(setParts, ", "), values
}

// scanRow сканирует одну строку (*sql.Row) в структуру entity.
// Работает с embedded structs (BaseEntity внутри Plant, Disease, ...).
func scanRow(row *sql.Row, entity any) error {
	scanners, err := buildScanners(entity)
	if err != nil {
		return err
	}
	if err := row.Scan(scanners...); err != nil {
		return err
	}
	return applyScanners(entity, scanners)
}

// scanRowTo сканирует строку из *sql.Rows в структуру entity.
func scanRowTo(rows *sql.Rows, entity any) error {
	// Получаем список колонок из результата запроса
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	scanners, err := buildScannersForColumns(entity, columns)
	if err != nil {
		return err
	}
	if err := rows.Scan(scanners...); err != nil {
		return err
	}
	return applyScannersForColumns(entity, scanners, columns)
}

// buildScanners создаёт slice указателей для Scan на основе структуры.
// Использует порядок полей в структуре (должен совпадать с SELECT *).
func buildScanners(entity any) ([]any, error) {
	fields := collectFields(entity)
	scanners := make([]any, len(fields))
	for i, f := range fields {
		scanners[i] = f.Addr().Interface()
	}
	return scanners, nil
}

// buildScannersForColumns создаёт slice указателей с учётом порядка колонок из БД.
func buildScannersForColumns(entity any, columns []string) ([]any, error) {
	fieldMap := buildFieldMap(entity)
	scanners := make([]any, len(columns))
	for i, col := range columns {
		if field, ok := fieldMap[col]; ok {
			scanners[i] = field.Addr().Interface()
		} else {
			// Колонка есть в БД, но нет в структуре — игнорируем
			var ignored any
			scanners[i] = &ignored
		}
	}
	return scanners, nil
}

// applyScanners — ничего не нужно, т.к. Scan пишет напрямую в поля через указатели.
func applyScanners(entity any, scanners []any) error {
	return nil
}

// applyScannersForColumns — аналогично, ничего не нужно.
func applyScannersForColumns(entity any, scanners []any, columns []string) error {
	return nil
}

// collectFields собирает все поля структуры (включая embedded) в плоский slice.
// Порядок совпадает с порядком в SELECT *.
func collectFields(entity any) []reflect.Value {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return collectFieldsFromValue(v)
}

// collectFieldsFromValue — рекурсивный сбор полей с поддержкой embedded.
func collectFieldsFromValue(v reflect.Value) []reflect.Value {
	var fields []reflect.Value
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		// Пропускаем неэкспортируемые поля
		if !field.IsExported() {
			continue
		}

		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		// Embedded struct (BaseEntity) — проваливаемся внутрь
		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			embedded := collectFieldsFromValue(fieldVal)
			fields = append(fields, embedded...)
			continue
		}

		// Обычное поле с тегом db
		if dbTag != "" {
			fields = append(fields, fieldVal)
		}
	}

	return fields
}

// buildFieldMap создаёт map "колонка БД" → reflect.Value поля.
func buildFieldMap(entity any) map[string]reflect.Value {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	result := make(map[string]reflect.Value)
	buildFieldMapFromValue(v, result)
	return result
}

// buildFieldMapFromValue — рекурсивное построение map с поддержкой embedded.
func buildFieldMapFromValue(v reflect.Value, result map[string]reflect.Value) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		if !field.IsExported() {
			continue
		}

		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		// Embedded struct — проваливаемся
		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			buildFieldMapFromValue(fieldVal, result)
			continue
		}

		if dbTag != "" {
			result[dbTag] = fieldVal
		}
	}
}