package app

import (
        "context"
        "database/sql"
        "fmt"
        "time"

        "fitoscout/backend/internal/logging"
)

// cleanupTables — таблицы с soft-delete (deleted_at), подлежащие автоочистке.
// Tombstones хранятся retention_days для дельта-синхронизации (ADR-008),
// затем физически удаляются.
var cleanupTables = []string{
        "plants",
        "diseases",
        "pests",
        "agrochemicals",
        "active_substances",
        "terminologies",
        "articles",
        "registry",
        "calendar",
        "library",
        "comments",
        "entity_links",
        "record_images",
}

// moduleTables — ключи модулей (совпадают с именами таблиц), для которых
// чистятся зависимые данные: EAV-значения и сквозной индекс ссылок.
var moduleTables = []string{
        "plants",
        "diseases",
        "pests",
        "agrochemicals",
        "active_substances",
        "terminologies",
        "articles",
        "registry",
        "calendar",
        "library",
        "comments",
}

// RunAutoCleanup — фоновая задача автоочистки (фича 4).
// Первый запуск — через cfg.FirstDelayMinutes после старта приложения,
// далее — каждые cfg.IntervalHours (по умолчанию раз в 24 часа).
// Запускается из Start() в горутине; останавливается через ctx.
func (a *App) RunAutoCleanup(ctx context.Context) error {
        cfg := a.cfg.Cleanup
        firstDelay := time.Duration(cfg.FirstDelayMinutes) * time.Minute
        interval := time.Duration(cfg.IntervalHours) * time.Hour

        a.logger.Info("автоочистка запланирована",
                logging.F("retention_days", cfg.RetentionDays),
                logging.F("interval_hours", cfg.IntervalHours),
                logging.F("first_delay_minutes", cfg.FirstDelayMinutes),
        )

        timer := time.NewTimer(firstDelay)
        defer timer.Stop()

        for {
                select {
                case <-ctx.Done():
                        a.logger.Info("автоочистка остановлена", logging.F("reason", "приложение завершается"))
                        return nil
                case <-timer.C:
                        start := time.Now()
                        total, err := a.cleanupOnce(ctx, cfg.RetentionDays)
                        if err != nil {
                                a.logger.Error("ошибка автоочистки",
                                        logging.F("error", err.Error()),
                                        logging.F("duration_ms", time.Since(start).Milliseconds()),
                                )
                        } else {
                                a.logger.Info("автоочистка завершена",
                                        logging.F("total_deleted", total),
                                        logging.F("duration_ms", time.Since(start).Milliseconds()),
                                )
                        }
                        timer.Reset(interval)
                }
        }
}

// cleanupOnce выполняет один проход автоочистки и возвращает общее число
// физически удалённых записей.
func (a *App) cleanupOnce(ctx context.Context, retentionDays int) (int, error) {
        cutoff := time.Now().AddDate(0, 0, -retentionDays)
        db := a.db.GetDB()
        total := 0

        a.logger.Info("автоочистка запущена",
                logging.F("cutoff", cutoff.UTC().Format(time.RFC3339)),
                logging.F("tables", len(cleanupTables)),
        )

        // 1. Физическое удаление tombstones старше cutoff.
        for _, table := range cleanupTables {
                // Имена таблиц — из статического списка cleanupTables, без пользовательского ввода.
                query := fmt.Sprintf("DELETE FROM %s WHERE deleted_at IS NOT NULL AND deleted_at < ?", table)
                res, err := db.ExecContext(ctx, query, cutoff)
                if err != nil {
                        return total, fmt.Errorf("таблица %s: %w", table, err)
                }
                deleted, err := res.RowsAffected()
                if err != nil {
                        return total, fmt.Errorf("таблица %s: не удалось получить число удалённых строк: %w", table, err)
                }
                if deleted > 0 {
                        a.logger.Info("записи удалены",
                                logging.F("table", table),
                                logging.F("deleted", deleted),
                        )
                }
                total += int(deleted)
        }

        // 2. Сиротские EAV-значения физически удалённых записей.
        orphans, err := a.cleanupOrphanAttributeValues(ctx, db)
        if err != nil {
                return total, fmt.Errorf("очистка attribute_values: %w", err)
        }
        if orphans > 0 {
                a.logger.Info("сиротские значения характеристик удалены",
                        logging.F("table", "attribute_values"),
                        logging.F("deleted", orphans),
                )
        }
        total += orphans

        // 3. Сиротские записи сквозного индекса ссылок (link_index без deleted_at).
        links, err := a.cleanupOrphanLinkIndex(ctx, db)
        if err != nil {
                return total, fmt.Errorf("очистка link_index: %w", err)
        }
        if links > 0 {
                a.logger.Info("сиротские записи индекса ссылок удалены",
                        logging.F("table", "link_index"),
                        logging.F("deleted", links),
                )
        }
        total += links

        return total, nil
}

// cleanupOrphanAttributeValues удаляет EAV-значения, чьи записи-владельцы
// уже физически удалены из модульных таблиц.
func (a *App) cleanupOrphanAttributeValues(ctx context.Context, db *sql.DB) (int, error) {
        total := 0
        for _, module := range moduleTables {
                query := "DELETE FROM attribute_values WHERE entity_module = ? " +
                        "AND entity_id NOT IN (SELECT id FROM " + module + ")"
                res, err := db.ExecContext(ctx, query, module)
                if err != nil {
                        return total, fmt.Errorf("модуль %s: %w", module, err)
                }
                deleted, err := res.RowsAffected()
                if err != nil {
                        return total, err
                }
                total += int(deleted)
        }
        return total, nil
}

// cleanupOrphanLinkIndex удаляет записи link_index, указывающие на
// несуществующие записи модулей.
func (a *App) cleanupOrphanLinkIndex(ctx context.Context, db *sql.DB) (int, error) {
        total := 0
        for _, module := range moduleTables {
                query := "DELETE FROM link_index WHERE " +
                        "(source_module = ? AND source_id NOT IN (SELECT id FROM " + module + ")) OR " +
                        "(target_module = ? AND target_id NOT IN (SELECT id FROM " + module + "))"
                res, err := db.ExecContext(ctx, query, module, module)
                if err != nil {
                        return total, fmt.Errorf("модуль %s: %w", module, err)
                }
                deleted, err := res.RowsAffected()
                if err != nil {
                        return total, err
                }
                total += int(deleted)
        }
        return total, nil
}