package storage

import (
	"fmt"
	"time"

	"fitoscout/backend/internal/config"
)

// Config — конфигурация хранилища MariaDB (задача #1).
// SQLite-специфика (Pragmas: journal_mode, synchronous, cache_size,
// busy_timeout, foreign_keys) удалена — ею управляет сервер MariaDB.
type Config struct {
	// DSN — user:pass@tcp(host:port)/db?parseTime=true&loc=UTC
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// FromAppConfig собирает конфигурацию хранилища из конфига приложения.
func FromAppConfig(cfg *config.Config) Config {
	return Config{
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetimeDuration(),
	}
}

// Validate проверяет параметры подключения.
func (c Config) Validate() error {
	if c.DSN == "" {
		return fmt.Errorf("DSN для подключения к MariaDB не указан")
	}
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns должно быть больше нуля")
	}
	if c.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns не может быть отрицательным")
	}
	if c.ConnMaxLifetime < 0 {
		return fmt.Errorf("conn_max_lifetime не может быть отрицательным")
	}
	return nil
}
