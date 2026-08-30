package storage

import (
        "context"
        "database/sql"
        "fmt"
        "sort"
        "strconv"
        "strings"
        "time"

        "fitoscout/backend/internal/logging"
)

// migration — одна встроенная SQL-миграция.
type migration struct {
        Version int
        Name    string
        Body    string
}

// Migrator применяет встроенные миграции к MariaDB.
// Состояние схемы хранится в таблице schema_migrations.
type Migrator struct {
        db     *sql.DB
        logger *logging.Logger
}

// NewMigrator создаёт мигратор.
func NewMigrator(db *sql.DB, logger *logging.Logger) *Migrator {
        return &Migrator{db: db, logger: logger}
}

// Migrate применяет все ещё не применённые миграции по порядку.
func (m *Migrator) Migrate(ctx context.Context) error {
        if err := m.ensureTable(ctx); err != nil {
                return err
        }

        migrations, err := loadMigrations()
        if err != nil {
                return err
        }

        applied, err := m.appliedVersions(ctx)
        if err != nil {
                return err
        }

        pending := 0
        for _, mig := range migrations {
                if applied[mig.Version] {
                        continue
                }
                if err := m.apply(ctx, mig); err != nil {
                        return fmt.Errorf("ошибка применения миграции %s: %w", mig.Name, err)
                }
                pending++
        }

        if pending == 0 {
                m.logger.Info("миграции не требуются",
                        logging.F("total", len(migrations)),
                        logging.F("applied", len(applied)),
                )
        } else {
                m.logger.Info("миграции применены",
                        logging.F("applied_now", pending),
                        logging.F("total", len(migrations)),
                )
        }
        return nil
}

// ensureTable создаёт служебную таблицу версий схемы.
func (m *Migrator) ensureTable(ctx context.Context) error {
        const query = `CREATE TABLE IF NOT EXISTS schema_migrations (
                version INT PRIMARY KEY,
                name VARCHAR(255) NOT NULL,
                applied_at BIGINT NOT NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
        if _, err := m.db.ExecContext(ctx, query); err != nil {
                return fmt.Errorf("не удалось создать таблицу schema_migrations: %w", err)
        }
        return nil
}

// loadMigrations читает и сортирует встроенные SQL-файлы (NNNNNN_name.up.sql).
func loadMigrations() ([]migration, error) {
        entries, err := MigrationsFS.ReadDir("migrations")
        if err != nil {
                return nil, fmt.Errorf("не удалось прочитать встроенные миграции: %w", err)
        }

        var out []migration
        for _, e := range entries {
                if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
                        continue
                }
                body, err := MigrationsFS.ReadFile("migrations/" + e.Name())
                if err != nil {
                        return nil, fmt.Errorf("не удалось прочитать миграцию %s: %w", e.Name(), err)
                }
                version, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
                if err != nil {
                        return nil, fmt.Errorf("некорректный номер миграции %s: %w", e.Name(), err)
                }
                out = append(out, migration{Version: version, Name: e.Name(), Body: string(body)})
        }
        sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
        return out, nil
}

// appliedVersions возвращает набор уже применённых версий.
func (m *Migrator) appliedVersions(ctx context.Context) (map[int]bool, error) {
        rows, err := m.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
        if err != nil {
                return nil, fmt.Errorf("не удалось получить список применённых миграций: %w", err)
        }
        defer rows.Close()

        applied := make(map[int]bool)
        for rows.Next() {
                var v int
                if err := rows.Scan(&v); err != nil {
                        return nil, fmt.Errorf("ошибка чтения schema_migrations: %w", err)
                }
                applied[v] = true
        }
        return applied, rows.Err()
}

// apply выполняет миграцию в транзакции и отмечает её применённой.
func (m *Migrator) apply(ctx context.Context, mig migration) error {
        tx, err := m.db.BeginTx(ctx, nil)
        if err != nil {
                return fmt.Errorf("не удалось открыть транзакцию: %w", err)
        }
        defer func() { _ = tx.Rollback() }()

        // Драйвер mysql по умолчанию не выполняет мульти-выражения,
        // поэтому файл миграции разбивается на отдельные выражения.
        for i, stmt := range splitStatements(mig.Body) {
                if _, err := tx.ExecContext(ctx, stmt); err != nil {
                        return fmt.Errorf("выражение #%d: %w", i+1, err)
                }
        }

        if _, err := tx.ExecContext(ctx,
                `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
                mig.Version, mig.Name, time.Now().UnixMilli(),
        ); err != nil {
                return fmt.Errorf("не удалось отметить миграцию применённой: %w", err)
        }

        if err := tx.Commit(); err != nil {
                return fmt.Errorf("не удалось зафиксировать транзакцию: %w", err)
        }

        m.logger.Info("миграция применена",
                logging.F("version", mig.Version),
                logging.F("name", mig.Name),
        )
        return nil
}

// splitStatements разбивает SQL-файл на отдельные выражения по ";".
// Пустые фрагменты и фрагменты только из комментариев отбрасываются.
func splitStatements(body string) []string {
        parts := strings.Split(body, ";")
        out := make([]string, 0, len(parts))
        for _, part := range parts {
                stmt := strings.TrimSpace(part)
                if stmt == "" || isCommentOnly(stmt) {
                        continue
                }
                out = append(out, stmt)
        }
        return out
}

// isCommentOnly сообщает, состоит ли фрагмент только из SQL-комментариев.
func isCommentOnly(s string) bool {
        for _, line := range strings.Split(s, "\n") {
                line = strings.TrimSpace(line)
                if line == "" || strings.HasPrefix(line, "--") {
                        continue
                }
                return false
        }
        return true
}