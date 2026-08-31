package handlers

import (
	"encoding/json"
	"net/http"

	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
	apperrors "fitoscout/backend/internal/errors"
)

// CommentsHandler обрабатывает CRUD запросы для модуля "Комментарии".
type CommentsHandler struct {
	repo repositories.Repository[domain.Comment]
}

// NewCommentsHandler создаёт новый обработчик для комментариев.
func NewCommentsHandler(repo repositories.Repository[domain.Comment]) *CommentsHandler {
	return &CommentsHandler{repo: repo}
}

// List — GET /api/v1/comments?limit=N&offset=M&module_key=X&entity_id=Y (доступно всем ролям)
func (h *CommentsHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Примечание: фильтрация по module_key и entity_id может быть добавлена в репозиторий
	// Для базовой реализации используем стандартный List
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

// GetByID — GET /api/v1/comments/{id} (доступно всем ролям)
func (h *CommentsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "comments")
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

// Create — POST /api/v1/comments (доступно всем ролям: web И android)
func (h *CommentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var entity domain.Comment
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
		return
	}

	// Валидация: текст комментария обязателен
	if entity.Text == "" {
		apperrors.WriteError(w, apperrors.Validation("текст комментария обязателен", nil))
		return
	}

	// Валидация: тип комментария
	validTypes := map[string]bool{"comment": true, "general": true, "task": true}
	if !validTypes[entity.Type] {
		apperrors.WriteError(w, apperrors.Validation("неподдерживаемый тип комментария (comment|general|task)", nil))
		return
	}

	// Валидация: статус комментария
	validStatuses := map[string]bool{"new": true, "in_progress": true, "done": true}
	if !validStatuses[entity.Status] {
		apperrors.WriteError(w, apperrors.Validation("неподдерживаемый статус (new|in_progress|done)", nil))
		return
	}

	// Валидация: module_key обязателен
	if entity.ModuleKey == "" {
		apperrors.WriteError(w, apperrors.Validation("module_key обязателен", nil))
		return
	}

	// Валидация: entity_id обязателен
	if entity.EntityID == "" {
		apperrors.WriteError(w, apperrors.Validation("entity_id обязателен", nil))
		return
	}

	if err := h.repo.Create(r.Context(), &entity); err != nil {
		apperrors.WriteError(w, err)
		return
	}

	apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/comments/{id} (доступно всем ролям)
func (h *CommentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "comments")
	if id == "" {
		apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
		return
	}

	var entity domain.Comment
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
		return
	}

	entity.SetID(id)

	// Валидация: текст комментария обязателен
	if entity.Text == "" {
		apperrors.WriteError(w, apperrors.Validation("текст комментария обязателен", nil))
		return
	}

	// Валидация: тип комментария
	validTypes := map[string]bool{"comment": true, "general": true, "task": true}
	if !validTypes[entity.Type] {
		apperrors.WriteError(w, apperrors.Validation("неподдерживаемый тип комментария (comment|general|task)", nil))
		return
	}

	// Валидация: статус комментария
	validStatuses := map[string]bool{"new": true, "in_progress": true, "done": true}
	if !validStatuses[entity.Status] {
		apperrors.WriteError(w, apperrors.Validation("неподдерживаемый статус (new|in_progress|done)", nil))
		return
	}

	if err := h.repo.Update(r.Context(), &entity); err != nil {
		apperrors.WriteError(w, err)
		return
	}

	apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/comments/{id} (доступно всем ролям)
func (h *CommentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "comments")
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

// RegisterRoutes регистрирует роуты для комментариев.
func (h *CommentsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/comments", h.List)
	mux.HandleFunc("GET /api/v1/comments/{id}", h.GetByID)
	mux.HandleFunc("POST /api/v1/comments", h.Create)
	mux.HandleFunc("PUT /api/v1/comments/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/comments/{id}", h.Delete)
}
