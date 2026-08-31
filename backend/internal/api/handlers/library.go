package handlers

import (
	"encoding/json"
	"net/http"

	"fitoscout/backend/internal/domain"
	"fitoscout/backend/internal/domain/repositories"
	apperrors "fitoscout/backend/internal/errors"
)

// LibraryHandler обрабатывает CRUD запросы для модуля "Библиотека".
type LibraryHandler struct {
	repo repositories.Repository[domain.LibraryItem]
}

// NewLibraryHandler создаёт новый обработчик для библиотеки.
func NewLibraryHandler(repo repositories.Repository[domain.LibraryItem]) *LibraryHandler {
	return &LibraryHandler{repo: repo}
}

// List — GET /api/v1/library?limit=N&offset=M (доступно всем ролям)
func (h *LibraryHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/library/{id} (доступно всем ролям)
func (h *LibraryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "library")
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

// Create — POST /api/v1/library (доступно всем ролям: web И android)
func (h *LibraryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var entity domain.LibraryItem
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
		return
	}

	// Валидация: название обязательно
	if entity.Title == "" {
		apperrors.WriteError(w, apperrors.Validation("название обязательно", nil))
		return
	}

	// Валидация: формат файла
	validFormats := map[string]bool{"pdf": true, "djvu": true, "mp4": true, "mkv": true}
	if !validFormats[entity.Format] {
		apperrors.WriteError(w, apperrors.Validation("неподдерживаемый формат файла (pdf|djvu|mp4|mkv)", nil))
		return
	}

	// Валидация: путь к файлу обязателен
	if entity.FilePath == "" {
		apperrors.WriteError(w, apperrors.Validation("путь к файлу обязателен", nil))
		return
	}

	if err := h.repo.Create(r.Context(), &entity); err != nil {
		apperrors.WriteError(w, err)
		return
	}

	apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/library/{id} (доступно всем ролям)
func (h *LibraryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "library")
	if id == "" {
		apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
		return
	}

	var entity domain.LibraryItem
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
		return
	}

	entity.SetID(id)

	// Валидация: название обязательно
	if entity.Title == "" {
		apperrors.WriteError(w, apperrors.Validation("название обязательно", nil))
		return
	}

	// Валидация: формат файла
	validFormats := map[string]bool{"pdf": true, "djvu": true, "mp4": true, "mkv": true}
	if !validFormats[entity.Format] {
		apperrors.WriteError(w, apperrors.Validation("неподдерживаемый формат файла (pdf|djvu|mp4|mkv)", nil))
		return
	}

	if err := h.repo.Update(r.Context(), &entity); err != nil {
		apperrors.WriteError(w, err)
		return
	}

	apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/library/{id} (доступно всем ролям)
func (h *LibraryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := extractID(r, "library")
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

// RegisterRoutes регистрирует роуты для библиотеки.
func (h *LibraryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/library", h.List)
	mux.HandleFunc("GET /api/v1/library/{id}", h.GetByID)
	mux.HandleFunc("POST /api/v1/library", h.Create)
	mux.HandleFunc("PUT /api/v1/library/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/library/{id}", h.Delete)
}
