package handlers

import (
        "encoding/json"
        "net/http"
        "time"

        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
        "fitoscout/backend/internal/errors"
)

// CalendarHandler обрабатывает HTTP-запросы к модулю "Календарь".
type CalendarHandler struct {
        repo repositories.Repository[domain.CalendarEvent]
}

// NewCalendarHandler создаёт новый хэндлер календаря.
func NewCalendarHandler(repo repositories.Repository[domain.CalendarEvent]) *CalendarHandler {
        return &CalendarHandler{repo: repo}
}

// List — GET /api/v1/calendar (доступно всем ролям).
func (h *CalendarHandler) List(w http.ResponseWriter, r *http.Request) {
        limit := 50
        offset := 0

        entities, err := h.repo.List(r.Context(), limit, offset)
        if err != nil {
                errors.WriteError(w, err)
                return
        }

        count, _ := h.repo.Count(r.Context())

        errors.WriteJSON(w, http.StatusOK, map[string]any{
                "data":   entities,
                "total":  count,
                "limit":  limit,
                "offset": offset,
        })
}

// GetByID — GET /api/v1/calendar/{id} (доступно всем ролям).
func (h *CalendarHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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

// Create — POST /api/v1/calendar (доступно всем ролям: web И android).
func (h *CalendarHandler) Create(w http.ResponseWriter, r *http.Request) {
        var entity domain.CalendarEvent
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация: название обязательно
        if entity.Title == "" {
                errors.WriteError(w, errors.Validation("название события обязательно", nil))
                return
        }

        // Валидация: дата события обязательна
        if entity.EventAt.IsZero() {
                errors.WriteError(w, errors.Validation("дата события обязательна", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/calendar/{id} (доступно всем ролям).
func (h *CalendarHandler) Update(w http.ResponseWriter, r *http.Request) {
        id := getURLParam(r, "id")
        if id == "" {
                errors.WriteError(w, errors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.CalendarEvent
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация: название обязательно
        if entity.Title == "" {
                errors.WriteError(w, errors.Validation("название события обязательно", nil))
                return
        }

        // Валидация: дата события обязательна
        if entity.EventAt.IsZero() {
                errors.WriteError(w, errors.Validation("дата события обязательна", nil))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/calendar/{id} (доступно всем ролям).
func (h *CalendarHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// MarkDone — POST /api/v1/calendar/{id}/done (отметить выполненным).
func (h *CalendarHandler) MarkDone(w http.ResponseWriter, r *http.Request) {
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

        now := time.Now()
        entity.DoneAt = &now

        if err := h.repo.Update(r.Context(), entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusOK, entity)
}

// RegisterRoutes регистрирует роуты для календаря.
func (h *CalendarHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/calendar", h.List)
        mux.HandleFunc("GET /api/v1/calendar/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/calendar", h.Create)
        mux.HandleFunc("PUT /api/v1/calendar/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/calendar/{id}", h.Delete)
        mux.HandleFunc("POST /api/v1/calendar/{id}/done", h.MarkDone)
}