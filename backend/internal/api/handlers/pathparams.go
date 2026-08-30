package handlers

import (
"net/http"
"strings"
)

// getURLParam извлекает параметр id из пути URL.
// Для путей вида /api/v1/calendar/{id} или /api/v1/calendar/{id}/done
// извлекает именно ID, а не последний сегмент.
func getURLParam(r *http.Request, paramName string) string {
        // Путь вида: /api/v1/calendar/123-456-789 или /api/v1/calendar/123-456-789/done
        parts := strings.Split(r.URL.Path, "/")
        if len(parts) < 5 {
                return ""
        }

        // parts[0] = "", parts[1] = "api", parts[2] = "v1", parts[3] = module, parts[4] = id or action
        // Для /api/v1/calendar/test-id → parts[4] = "test-id"
        // Для /api/v1/calendar/test-id/done → parts[4] = "test-id", parts[5] = "done"

        id := parts[4]

        // Если это составной путь с действием (например, /done), проверяем следующий сегмент
        if len(parts) >= 6 && parts[5] == "done" {
                return id
        }

        // Если это простой путь /api/v1/module/id, но id пустой (путь заканчивается на /)
        if id == "" {
                return ""
        }

        // Возвращаем ID (четвёртый сегмент после разбиения)
        return id
}