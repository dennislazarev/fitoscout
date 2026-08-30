package storage

import (
        "context"
        "database/sql"
        "time"

        "github.com/google/uuid"
)

// Queryer — общий интерфейс исполнителя запросов (*sql.DB или *sql.Tx).
// Используется репозиториями модулей (задачи #4 и #5).
type Queryer interface {
        ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
        QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
        QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewID генерирует новый первичный ключ — UUIDv7
// (соглашение MASTER.md, раздел 8.3).
func NewID() string {
        return uuid.Must(uuid.NewV7()).String()
}

// NowMillis возвращает текущее время в миллисекундах Unix
// (соглашение MASTER.md, раздел 8.4).
func NowMillis() int64 {
        return time.Now().UnixMilli()
}

// IsNotFound сообщает, является ли ошибкой «запись не найдена».
func IsNotFound(err error) bool {
        return err == sql.ErrNoRows
}