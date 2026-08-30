package handlers

import (
        "encoding/json"
        "net/http"

        apperrors "fitoscout/backend/internal/errors"
        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// CalendarHandler обрабатывает CRUD запросы для модуля "Календарь".
type CalendarHandler struct {
        repo repositories.Repository[domain.CalendarEvent]
}

// NewCalendarHandler создаёт новый обработчик для календаря.
func NewCalendarHandler(repo repositories.Repository[domain.CalendarEvent]) *CalendarHandler {
        return &CalendarHandler{repo: repo}
}

// List — GET /api/v1/calendar?limit=N&offset=M (доступно всем ролям)
func (h *CalendarHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/calendar/{id} (доступно всем ролям)
func (h *CalendarHandler) GetByID(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "calendar")
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

// Create — POST /api/v1/calendar (доступно всем ролям: web И android)
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
        var entity domain.CalendarEvent
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация: название события обязательно
        if entity.Title == "" {
                apperrors.WriteError(w, apperrors.Validation("название события обязательно", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/calendar/{id} (доступно всем ролям)
func (h *CalendarHandler) Update(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "calendar")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.CalendarEvent
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация: название события обязательно
        if entity.Title == "" {
                apperrors.WriteError(w, apperrors.Validation("название события обязательно", nil))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/calendar/{id} (доступно всем ролям)
func (h *CalendarHandler) Delete(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "calendar")
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

// RegisterRoutes регистрирует роуты для календаря.
func (h *CalendarHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/calendar", h.List)
        mux.HandleFunc("GET /api/v1/calendar/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/calendar", h.Create)
        mux.HandleFunc("PUT /api/v1/calendar/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/calendar/{id}", h.Delete)
}