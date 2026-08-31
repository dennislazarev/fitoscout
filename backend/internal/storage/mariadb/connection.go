// Package mariadb — низкоуровневое подключение к MariaDB
// (драйвер github.com/go-sql-driver/mysql, задача #1).
package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Options — параметры подключения и пула.
type Options struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Open создаёт пул подключений и проверяет доступность сервера
// с повторными попытками (сервер может стартовать дольше приложения).
func Open(ctx context.Context, opts Options, attempts int, retryDelay time.Duration) (*sql.DB, error) {
	db, err := sql.Open("mysql", opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("не удалось инициализировать драйвер MariaDB: %w", err)
	}

	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)

	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for i := 1; i <= attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = db.PingContext(pingCtx)
		cancel()
		if lastErr == nil {
			return db, nil
		}
		if i < attempts {
			time.Sleep(retryDelay)
		}
	}

	_ = db.Close()
	return nil, fmt.Errorf("не удалось подключиться к MariaDB после %d попыток: %w", attempts, lastErr)
}

// dsnPasswordRe находит фрагмент "user:password@" в начале DSN.
// Исправленная регулярка: захватывает всё до последнего @ перед tcp( или unix(
var dsnPasswordRe = regexp.MustCompile(`^([^:/]+):(.+)@(tcp\(|unix\()`)

// SanitizeDSN маскирует пароль в DSN для безопасного логирования:
// "fitoscout:secret@tcp(...)" → "fitoscout:***@tcp(...)".
// Корректно работает с паролями, содержащими символ @.
func SanitizeDSN(dsn string) string {
	return dsnPasswordRe.ReplaceAllString(dsn, "$1:***@$3")
}
