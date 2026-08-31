package handlers

import (
	"encoding/json"
	"net/http"

	"fitoscout/backend/internal/auth"
	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
	apperrors "fitoscout/backend/internal/errors"
)

// TerminologiesHandler обрабатывает CRUD запросы для модуля "Терминология".
type TerminologiesHandler struct {
	repo repositories.Repository[domain.Terminology]
}

// NewTerminologiesHandler создаёт новый обработчик для терминологии.
func NewTerminologiesHandler(repo repositories.Repository[domain.Terminology]) *TerminologiesHandler {
	return &TerminologiesHandler{repo: repo}
}

// List — GET /api/v1/terminologies?limit=N&offset=M
func (h *TerminologiesHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/terminologies/{id}
func (h *TerminologiesHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "terminologies")
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

// Create — POST /api/v1/terminologies (только web)
func (h *TerminologiesHandler) Create(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFromContext(r.Context())
	if role != auth.RoleWeb {
		apperrors.WriteError(w, apperrors.Forbidden("создание доступно только админке"))
		return
	}

	var entity domain.Terminology
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
		return
	}

	// Валидация
	if entity.Name == "" {
		apperrors.WriteError(w, apperrors.Validation("название термина обязательно", nil))
		return
	}

	if err := h.repo.Create(r.Context(), &entity); err != nil {
		apperrors.WriteError(w, err)
		return
	}

	apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/terminologies/{id} (только web)
func (h *TerminologiesHandler) Update(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFromContext(r.Context())
	if role != auth.RoleWeb {
		apperrors.WriteError(w, apperrors.Forbidden("обновление доступно только админке"))
		return
	}

	id := extractID(r, "terminologies")
	if id == "" {
		apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
		return
	}

	var entity domain.Terminology
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

// Delete — DELETE /api/v1/terminologies/{id} (только web)
func (h *TerminologiesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFromContext(r.Context())
	if role != auth.RoleWeb {
		apperrors.WriteError(w, apperrors.Forbidden("удаление доступно только админке"))
		return
	}

	id := extractID(r, "terminologies")
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

// RegisterRoutes регистрирует роуты для терминологии.
func (h *TerminologiesHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/terminologies", h.List)
	mux.HandleFunc("GET /api/v1/terminologies/{id}", h.GetByID)
	mux.HandleFunc("POST /api/v1/terminologies", h.Create)
	mux.HandleFunc("PUT /api/v1/terminologies/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/terminologies/{id}", h.Delete)
}
