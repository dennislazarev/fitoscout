package handlers

import (
        "encoding/json"
        "net/http"

        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
        "fitoscout/backend/internal/errors"
)

// CommentsHandler обрабатывает HTTP-запросы к модулю "Комментарии".
type CommentsHandler struct {
        repo repositories.Repository[domain.Comment]
}

// NewCommentsHandler создаёт новый хэндлер комментариев.
func NewCommentsHandler(repo repositories.Repository[domain.Comment]) *CommentsHandler {
        return &CommentsHandler{repo: repo}
}

// List — GET /api/v1/comments (доступно всем ролям).
// Поддерживает фильтрацию по module_key и entity_id.
func (h *CommentsHandler) List(w http.ResponseWriter, r *http.Request) {
        limit := 50
        offset := 0

        moduleKey := r.URL.Query().Get("module_key")
        entityID := r.URL.Query().Get("entity_id")

        var entities []domain.Comment
        var err error
		var count int64

        // Если указаны module_key и entity_id — фильтруем по ним
        if moduleKey != "" && entityID != "" {
                // Примечание: для полноценной фильтрации нужен метод FindByEntity
                // Пока используем общий List, фильтрация на клиенте или доработка репозитория
                entities, err = h.repo.List(r.Context(), limit, offset)
                if err != nil {
                        errors.WriteError(w, err)
                        return
                }

                // Фильтрация на уровне хэндлера (временное решение)
                filtered := make([]domain.Comment, 0)
                for _, c := range entities {
                        if c.ModuleKey == moduleKey && c.EntityID == entityID {
                                filtered = append(filtered, c)
                        }
                }
                entities = filtered
				count = int64(len(entities))
        } else {
                entities, err = h.repo.List(r.Context(), limit, offset)
                if err != nil {
                        errors.WriteError(w, err)
                        return
                }
				count, _ = h.repo.Count(r.Context())
        }

        errors.WriteJSON(w, http.StatusOK, map[string]any{
                "data":   entities,
                "total":  count,
                "limit":  limit,
                "offset": offset,
        })
}

// GetByID — GET /api/v1/comments/{id} (доступно всем ролям).
func (h *CommentsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
        id := getURLParam(r, "id")
        if id == "" {
                errors.WriteError(w, errors.BadRequest("ID обязателен"))
                return
        }

        entity, err := h.repo.GetByID(r.Context(), id)
        if err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusOK, entity)
}

// Create — POST /api/v1/comments (доступно всем ролям: web И android).
func (h *CommentsHandler) Create(w http.ResponseWriter, r *http.Request) {
        var entity domain.Comment
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация: текст обязателен
        if entity.Text == "" {
                errors.WriteError(w, errors.Validation("текст комментария обязателен", nil))
                return
        }

        // Валидация: module_key обязателен
        if entity.ModuleKey == "" {
                errors.WriteError(w, errors.Validation("module_key обязателен", nil))
                return
        }

        // Валидация: entity_id обязателен
        if entity.EntityID == "" {
                errors.WriteError(w, errors.Validation("entity_id обязателен", nil))
                return
        }

        // Валидация: тип комментария
        validTypes := map[string]bool{"comment": true, "general": true, "task": true}
        if entity.Type != "" && !validTypes[entity.Type] {
                errors.WriteError(w, errors.Validation("недопустимый тип комментария", nil))
                return
        }

        // Устанавливаем тип по умолчанию
        if entity.Type == "" {
                entity.Type = "comment"
        }

        // Устанавливаем статус по умолчанию
        if entity.Status == "" {
                entity.Status = "new"
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/comments/{id} (доступно всем ролям).
func (h *CommentsHandler) Update(w http.ResponseWriter, r *http.Request) {
        id := getURLParam(r, "id")
        if id == "" {
                errors.WriteError(w, errors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.Comment
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация: текст обязателен
        if entity.Text == "" {
                errors.WriteError(w, errors.Validation("текст комментария обязателен", nil))
                return
        }

        // Валидация: статус
        validStatuses := map[string]bool{"new": true, "in_progress": true, "done": true}
        if entity.Status != "" && !validStatuses[entity.Status] {
                errors.WriteError(w, errors.Validation("недопустимый статус комментария", nil))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/comments/{id} (доступно всем ролям).
func (h *CommentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
        id := getURLParam(r, "id")
        if id == "" {
                errors.WriteError(w, errors.BadRequest("ID обязателен"))
                return
        }

        if err := h.repo.SoftDelete(r.Context(), id); err != nil {
                errors.WriteError(w, err)
                return
        }

        w.WriteHeader(http.StatusNoContent)
}

// RegisterRoutes регистрирует роуты для комментариев.
func (h *CommentsHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/comments", h.List)
        mux.HandleFunc("GET /api/v1/comments/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/comments", h.Create)
        mux.HandleFunc("PUT /api/v1/comments/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/comments/{id}", h.Delete)
}