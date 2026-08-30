package handlers

import (
        "encoding/json"
        "net/http"
        "strconv"
        "strings"

        "fitoscout/backend/internal/auth"
        apperrors "fitoscout/backend/internal/errors"
        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// parseInt парсит строку в int, возвращает 0 при ошибке.
func parseInt(s string) int {
        n, err := strconv.Atoi(s)
        if err != nil {
                return 0
        }
        return n
}

// extractID извлекает ID из пути запроса (для Go 1.19 без PathValue).
func extractID(r *http.Request, prefix string) string {
        // Путь вида /api/v1/plants/{id}
        path := r.URL.Path
        idx := strings.LastIndex(path, "/")
        if idx == -1 || idx >= len(path)-1 {
                return ""
        }
        return path[idx+1:]
}

// PlantsHandler обрабатывает CRUD запросы для модуля "Растения".
type PlantsHandler struct {
        repo repositories.Repository[domain.Plant]
}

// NewPlantsHandler создаёт новый обработчик для растений.
func NewPlantsHandler(repo repositories.Repository[domain.Plant]) *PlantsHandler {
        return &PlantsHandler{repo: repo}
}

// List — GET /api/v1/plants?limit=N&offset=M
func (h *PlantsHandler) List(w http.ResponseWriter, r *http.Request) {
        // Парсинг query параметров
        limit := 50
        offset := 0

        if l := r.URL.Query().Get("limit"); l != "" {
                if n := parseInt(l); n > 0 {
                        limit = n
                }
        }
        if o := r.URL.Query().Get("offset"); o != "" {
                if n := parseInt(o); n >= 0 {
                        offset = n
                }
        }

        entities, err := h.repo.List(r.Context(), limit, offset)
        if err != nil {
                apperrors.WriteError(w, err)
                return
        }

        count, _ := h.repo.Count(r.Context())

        apperrors.WriteJSON(w, http.StatusOK, map[string]any{
                "data":   entities,
                "total":  count,
                "limit":  limit,
                "offset": offset,
        })
}

// GetByID — GET /api/v1/plants/{id}
func (h *PlantsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "plants")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        entity, err := h.repo.GetByID(r.Context(), id)
        if err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Create — POST /api/v1/plants (только web)
func (h *PlantsHandler) Create(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("создание доступно только админке"))
                return
        }

        var entity domain.Plant
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация
        if entity.Name == "" {
                apperrors.WriteError(w, apperrors.Validation("название растения обязательно", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/plants/{id} (только web)
func (h *PlantsHandler) Update(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("обновление доступно только админке"))
                return
        }

        id := extractID(r, "plants")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.Plant
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/plants/{id} (только web)
func (h *PlantsHandler) Delete(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("удаление доступно только админке"))
                return
        }

        id := extractID(r, "plants")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        if err := h.repo.SoftDelete(r.Context(), id); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes регистрирует роуты для растений.
func (h *PlantsHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/plants", h.List)
        mux.HandleFunc("GET /api/v1/plants/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/plants", h.Create)
        mux.HandleFunc("PUT /api/v1/plants/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/plants/{id}", h.Delete)
}