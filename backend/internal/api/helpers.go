package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"fitoscout/backend/internal/errors"
)

// writeError записывает ошибку в формате JSON
func writeError(w http.ResponseWriter, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())
	json.NewEncoder(w).Encode(err.Envelope())
}

// extractID извлекает ID из пути запроса.
// Работает корректно как при вызове через ServeMux (продакшн), так и напрямую из тестов.
func extractID(r *http.Request, module string) string {
	// Приоритет 1: Go 1.22+ ServeMux PathValue (работает, если запрос прошел через ServeMux)
	if id := r.PathValue("id"); id != "" {
		return id
	}

	// Приоритет 2: Интеллектуальный ручной парсинг пути (работает в тестах и при прямом вызове)
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		return ""
	}

	parts := strings.Split(path, "/")

	// Ищем сегмент с именем модуля (учитываем единственное и множественное число: "library" или "libraries")
	// Например: для пути "api/v1/library/lib-1/stream" мы найдем "library" на индексе 2
	for i := 0; i < len(parts); i++ {
		if parts[i] == module || parts[i] == module+"s" {
			// ID должен идти сразу после названия модуля
			if i+1 < len(parts) {
				return parts[i+1]
			}
			return "" // Модуль найден, но ID после него нет
		}
	}

	// Fallback: Если название модуля не найдено в пути (например, префикс был удален
	// через http.StripPrefix или в тесте передан путь сразу с ID типа "/lib-1/stream"),
	// предполагаем, что путь начинается сразу с ID. Берем первый сегмент.
	return parts[0]
}
