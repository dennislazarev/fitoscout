package handlers

import (
        "encoding/json"
        "net/http"
        "regexp"

        "fitoscout/backend/internal/auth"
        apperrors "fitoscout/backend/internal/errors"
        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// ActiveSubstancesHandler обрабатывает CRUD запросы для модуля "Действующие вещества".
type ActiveSubstancesHandler struct {
        repo repositories.Repository[domain.ActiveSubstance]
}

// NewActiveSubstancesHandler создаёт новый обработчик для действующих веществ.
func NewActiveSubstancesHandler(repo repositories.Repository[domain.ActiveSubstance]) *ActiveSubstancesHandler {
        return &ActiveSubstancesHandler{repo: repo}
}

// casRegex проверяет формат CAS номера (XXXX-XX-X).
var casRegex = regexp.MustCompile(`^\d{2,7}-\d{2}-\d$`)

// List — GET /api/v1/active-substances?limit=N&offset=M
func (h *ActiveSubstancesHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/active-substances/{id}
func (h *ActiveSubstancesHandler) GetByID(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "active-substances")
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

// Create — POST /api/v1/active-substances (только web)
func (h *ActiveSubstancesHandler) Create(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("создание доступно только админке"))
                return
        }

        var entity domain.ActiveSubstance
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация
        if entity.Name == "" {
                apperrors.WriteError(w, apperrors.Validation("название вещества обязательно", nil))
                return
        }

        // Валидация CAS номера (если указан)
        if entity.CAS != nil && !casRegex.MatchString(*entity.CAS) {
                apperrors.WriteError(w, apperrors.Validation("неверный формат CAS номера (должен быть XXXX-XX-X)", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/active-substances/{id} (только web)
func (h *ActiveSubstancesHandler) Update(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("обновление доступно только админке"))
                return
        }

        id := extractID(r, "active-substances")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.ActiveSubstance
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация CAS номера (если указан)
        if entity.CAS != nil && !casRegex.MatchString(*entity.CAS) {
                apperrors.WriteError(w, apperrors.Validation("неверный формат CAS номера (должен быть XXXX-XX-X)", nil))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/active-substances/{id} (только web)
func (h *ActiveSubstancesHandler) Delete(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("удаление доступно только админке"))
                return
        }

        id := extractID(r, "active-substances")
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

// RegisterRoutes регистрирует роуты для действующих веществ.
func (h *ActiveSubstancesHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/active-substances", h.List)
        mux.HandleFunc("GET /api/v1/active-substances/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/active-substances", h.Create)
        mux.HandleFunc("PUT /api/v1/active-substances/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/active-substances/{id}", h.Delete)
}