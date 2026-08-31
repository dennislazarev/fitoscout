package storage

import "embed"

// MigrationsFS — SQL-миграции, встроенные в бинарник (ADR-002: один бинарник,
// миграции применяются автоматически при старте).
// Файлы лежат в internal/storage/migrations и именуются NNNNNN_name.up.sql.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
