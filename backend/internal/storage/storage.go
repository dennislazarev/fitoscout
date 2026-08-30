// Package storage — доступ к MariaDB и применение миграций.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"fitoscout/backend/internal/logging"
	"fitoscout/backend/internal/storage/mariadb"
)

const (
	connectAttempts   = 30
	connectRetryDelay = 2 * time.Second
)

// DB — обёртка над пулом подключений MariaDB.
type DB struct {
	db     *sql.DB        // ← возвращаем обычный sql.DB
	cfg    Config
	logger *logging.Logger
}

// Open подключается к MariaDB и проверяет доступность сервера.
func Open(ctx context.Context, cfg Config, logger *logging.Logger) (*DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	db, err := mariadb.Open(ctx, mariadb.Options{
		DSN:             cfg.DSN,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
	}, connectAttempts, connectRetryDelay)
	if err != nil {
		logger.Error("ошибка подключения к БД",
			logging.F("error", err.Error()),
			logging.F("dsn", mariadb.SanitizeDSN(cfg.DSN)),
		)
		return nil, err
	}

	logger.Info("подключение к MariaDB установлено",
		logging.F("dsn", mariadb.SanitizeDSN(cfg.DSN)),
		logging.F("max_open_conns", cfg.MaxOpenConns),
		logging.F("max_idle_conns", cfg.MaxIdleConns),
	)

	return &DB{db: db, cfg: cfg, logger: logger}, nil
}

// GetDB возвращает базовый *sql.DB.
func (d *DB) GetDB() *sql.DB {
	return d.db
}

// Ping проверяет доступность сервера БД.
func (d *DB) Ping(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return fmt.Errorf("MariaDB недоступна: %w", err)
	}
	return nil
}

// Close закрывает пул подключений.
func (d *DB) Close() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия пула подключений: %w", err)
	}
	d.logger.Info("пул подключений к MariaDB закрыт")
	return nil
}