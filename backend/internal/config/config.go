// Package config загружает и валидирует TOML-конфигурацию fitoscoutd (ADR: TOML).
package config

import (
        "fmt"
        "os"
        "time"

        toml "github.com/pelletier/go-toml/v2"
)

// Config — корневая структура конфигурации.
type Config struct {
		Server   ServerConfig   `toml:"server"`
		TLS      TLSConfig      `toml:"tls"`
		Auth     AuthConfig     `toml:"auth"`
		Roles    RolesConfig    `toml:"roles"`      
		Database DatabaseConfig `toml:"database"`
		Cleanup  CleanupConfig  `toml:"cleanup"`
		Media    MediaConfig    `toml:"media"`
		Logging  LoggingConfig  `toml:"logging"`
}

// ServerConfig — сетевые параметры HTTP-сервера.
type ServerConfig struct {
        Host               string `toml:"host"`
        Port               int    `toml:"port"`
        ReadTimeoutSec     int    `toml:"read_timeout_sec"`
        WriteTimeoutSec    int    `toml:"write_timeout_sec"`
        ShutdownTimeoutSec int    `toml:"shutdown_timeout_sec"`
}

// TLSConfig — пути к сертификатам mTLS (ADR-006).
type TLSConfig struct {
        CertFile    string `toml:"cert_file"`
        KeyFile     string `toml:"key_file"`
        CAFile      string `toml:"ca_file"`
        RevokedFile string `toml:"revoked_file"`
}

// AuthConfig — параметры проверки клиентов.
type AuthConfig struct {
	ClientHeader    string `toml:"client_header"`
	RateLimitPerMin int    `toml:"rate_limit_per_min"`
}

// RolesConfig — CN сертификатов для определения ролей (ADR-006).
type RolesConfig struct {
	WebCN     string `toml:"web_cn"`      // CN для ПК-админки
	AndroidCN string `toml:"android_cn"`  // CN для Android-клиента
}

// DatabaseConfig — параметры подключения к MariaDB (задача #1).
// SQLite-специфика (Path, JournalMode, Synchronous, CacheSizeKB,
// BusyTimeoutMs, ForeignKeys) удалена.
type DatabaseConfig struct {
        // DSN — строка подключения:
        // user:pass@tcp(host:port)/db?parseTime=true&loc=UTC
        DSN             string `toml:"dsn"`
        MaxOpenConns    int    `toml:"max_open_conns"`
        MaxIdleConns    int    `toml:"max_idle_conns"`
        ConnMaxLifetime int    `toml:"conn_max_lifetime"` // секунды
}

// ConnMaxLifetimeDuration возвращает conn_max_lifetime как time.Duration.
func (d DatabaseConfig) ConnMaxLifetimeDuration() time.Duration {
        return time.Duration(d.ConnMaxLifetime) * time.Second
}

// CleanupConfig — параметры автоочистки soft-deleted записей (ADR-008).
type CleanupConfig struct {
        Enabled           bool `toml:"enabled"`
        RetentionDays     int  `toml:"retention_days"`      // сколько дней хранить tombstones
        IntervalHours     int  `toml:"interval_hours"`      // периодичность, по умолчанию 24
        FirstDelayMinutes int  `toml:"first_delay_minutes"` // первый запуск после старта
}

// MediaConfig — пути к медиафайлам (ADR-009).
type MediaConfig struct {
        BasePath    string `toml:"base_path"`    // /volume1/Fitoscout_project/media
        LibraryPath string `toml:"library_path"` // /volume2/Библиотека
}

// LoggingConfig — параметры логгера.
type LoggingConfig struct {
	Level     string `toml:"level"`       // debug | info | warn | error
	Output    string `toml:"output"`      // stdout или путь к файлу
	MaxSizeMB int    `toml:"max_size_mb"` // макс. размер файла перед ротацией
	MaxFiles  int    `toml:"max_files"`   // сколько старых файлов хранить
}

// Load читает TOML-файл, применяет дефолты и валидирует результат.
func Load(path string) (*Config, error) {
        data, err := os.ReadFile(path)
        if err != nil {
                return nil, fmt.Errorf("не удалось прочитать файл конфига %s: %w", path, err)
        }

        cfg := &Config{}
        if err := toml.Unmarshal(data, cfg); err != nil {
                return nil, fmt.Errorf("ошибка разбора TOML в файле %s: %w", path, err)
        }

        cfg.applyDefaults()

        if err := cfg.Validate(); err != nil {
                return nil, fmt.Errorf("некорректная конфигурация %s: %w", path, err)
        }
        return cfg, nil
}

// applyDefaults подставляет значения по умолчанию для незаполненных полей.
func (c *Config) applyDefaults() {
        if c.Server.Host == "" {
                c.Server.Host = "0.0.0.0"
        }
        if c.Server.Port == 0 {
                c.Server.Port = 8443
        }
        if c.Server.ReadTimeoutSec == 0 {
                c.Server.ReadTimeoutSec = 30
        }
        if c.Server.WriteTimeoutSec == 0 {
                c.Server.WriteTimeoutSec = 30
        }
        if c.Server.ShutdownTimeoutSec == 0 {
                c.Server.ShutdownTimeoutSec = 30
        }

        if c.TLS.CertFile == "" {
                c.TLS.CertFile = "/volume1/Fitoscout_project/certs/server.crt"
        }
        if c.TLS.KeyFile == "" {
                c.TLS.KeyFile = "/volume1/Fitoscout_project/certs/server.key"
        }
        if c.TLS.CAFile == "" {
                c.TLS.CAFile = "/volume1/Fitoscout_project/certs/ca.crt"
        }
        if c.TLS.RevokedFile == "" {
                c.TLS.RevokedFile = "/volume1/Fitoscout_project/certs/revoked.txt"
        }

        if c.Auth.ClientHeader == "" {
                c.Auth.ClientHeader = "X-Fitoscout-Client"
        }
        if c.Auth.RateLimitPerMin == 0 {
                c.Auth.RateLimitPerMin = 120
        }

        // MariaDB: дефолтный DSN для локальной разработки.
        if c.Database.DSN == "" {
                c.Database.DSN = "fitoscout:password@tcp(127.0.0.1:3306)/fitoscout?parseTime=true&loc=UTC"
        }
        if c.Database.MaxOpenConns == 0 {
                c.Database.MaxOpenConns = 10
        }
        if c.Database.MaxIdleConns == 0 {
                c.Database.MaxIdleConns = 5
        }
        if c.Database.ConnMaxLifetime == 0 {
                c.Database.ConnMaxLifetime = 3600
        }

        if c.Cleanup.RetentionDays == 0 {
                c.Cleanup.RetentionDays = 30
        }
        if c.Cleanup.IntervalHours == 0 {
                c.Cleanup.IntervalHours = 24
        }
        if c.Cleanup.FirstDelayMinutes == 0 {
                c.Cleanup.FirstDelayMinutes = 5
        }

        if c.Media.BasePath == "" {
                c.Media.BasePath = "/volume1/Fitoscout_project/media"
        }
        if c.Media.LibraryPath == "" {
                c.Media.LibraryPath = "/volume2/Библиотека"
        }

        if c.Logging.Level == "" {
			c.Logging.Level = "info"
		}
		if c.Logging.Output == "" {
			c.Logging.Output = "stdout"
		}
		if c.Logging.MaxSizeMB == 0 {
			c.Logging.MaxSizeMB = 10
		}
		if c.Logging.MaxFiles == 0 {
			c.Logging.MaxFiles = 5
		}
		if c.Roles.WebCN == "" {
			c.Roles.WebCN = "fitoscout-web-admin"
		}
		if c.Roles.AndroidCN == "" {
			c.Roles.AndroidCN = "fitoscout-android-client"
		}
}

// Validate проверяет критичные параметры конфигурации.
func (c *Config) Validate() error {
        if c.Server.Port <= 0 || c.Server.Port > 65535 {
                return fmt.Errorf("server.port: некорректный порт %d", c.Server.Port)
        }
        if c.Database.DSN == "" {
                return fmt.Errorf("database.dsn: не указан DSN для подключения к MariaDB")
        }
        if c.Database.MaxOpenConns <= 0 {
                return fmt.Errorf("database.max_open_conns: должно быть больше нуля")
        }
        if c.Database.MaxIdleConns < 0 {
                return fmt.Errorf("database.max_idle_conns: не может быть отрицательным")
        }
        if c.Cleanup.Enabled {
                if c.Cleanup.RetentionDays <= 0 {
                        return fmt.Errorf("cleanup.retention_days: должно быть больше нуля")
                }
                if c.Cleanup.IntervalHours <= 0 {
                        return fmt.Errorf("cleanup.interval_hours: должно быть больше нуля")
                }
                if c.Cleanup.FirstDelayMinutes < 0 {
                        return fmt.Errorf("cleanup.first_delay_minutes: не может быть отрицательным")
                }
        }
        return nil
}