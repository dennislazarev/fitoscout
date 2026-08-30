package handlers

import (
        "encoding/json"
        "net/http"

        "fitoscout/backend/internal/auth"
        apperrors "fitoscout/backend/internal/errors"
        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// AgrochemicalsHandler обрабатывает CRUD запросы для модуля "Агрохимия".
type AgrochemicalsHandler struct {
        repo repositories.Repository[domain.Agrochemical]
}

// NewAgrochemicalsHandler создаёт новый обработчик для агрохимии.
func NewAgrochemicalsHandler(repo repositories.Repository[domain.Agrochemical]) *AgrochemicalsHandler {
        return &AgrochemicalsHandler{repo: repo}
}

// validForms — допустимые формы препарата.
var validForms = map[string]bool{
        "SP": true, "WG": true, "EC": true, "SC": true,
        "SL": true, "WP": true, "GR": true, "DF": true,
        "MD": true, "CS": true, "ME": true, "OD": true,
        "SE": true, "EW": true, "AO": true, "OF": true,
        "PP": true, "TP": true, "DS": true, "WS": true,
        "LS": true, "FS": true, "GS": true, "BR": true,
        "MT": true, "SV": true, "VV": true, "CC": true,
}

// List — GET /api/v1/agrochemicals?limit=N&offset=M
func (h *AgrochemicalsHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/agrochemicals/{id}
func (h *AgrochemicalsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "agrochemicals")
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

// Create — POST /api/v1/agrochemicals (только web)
func (h *AgrochemicalsHandler) Create(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("создание доступно только админке"))
                return
        }

        var entity domain.Agrochemical
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация
        if entity.Name == "" {
                apperrors.WriteError(w, apperrors.Validation("название препарата обязательно", nil))
                return
        }

        // Валидация формы препарата (если указана)
        if entity.Form != nil && !validForms[*entity.Form] {
                apperrors.WriteError(w, apperrors.Validation("неверная форма препарата", map[string]any{
                        "допустимые значения": []string{"SP", "WG", "EC", "SC", "SL", "WP", "GR", "DF"},
                }))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/agrochemicals/{id} (только web)
func (h *AgrochemicalsHandler) Update(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("обновление доступно только админке"))
                return
        }

        id := extractID(r, "agrochemicals")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.Agrochemical
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация формы препарата (если указана)
        if entity.Form != nil && !validForms[*entity.Form] {
                apperrors.WriteError(w, apperrors.Validation("неверная форма препарата", map[string]any{
                        "допустимые значения": []string{"SP", "WG", "EC", "SC", "SL", "WP", "GR", "DF"},
                }))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/agrochemicals/{id} (только web)
func (h *AgrochemicalsHandler) Delete(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("удаление доступно только админке"))
                return
        }

        id := extractID(r, "agrochemicals")
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

// RegisterRoutes регистрирует роуты для агрохимии.
func (h *AgrochemicalsHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/agrochemicals", h.List)
        mux.HandleFunc("GET /api/v1/agrochemicals/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/agrochemicals", h.Create)
        mux.HandleFunc("PUT /api/v1/agrochemicals/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/agrochemicals/{id}", h.Delete)
}