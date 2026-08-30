package handlers

import (
        "encoding/json"
        "net/http"

        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
        "fitoscout/backend/internal/errors"
)

// RegistryHandler обрабатывает HTTP-запросы к модулю "Реестр питомника".
type RegistryHandler struct {
        repo repositories.Repository[domain.RegistryItem]
}

// NewRegistryHandler создаёт новый хэндлер реестра.
func NewRegistryHandler(repo repositories.Repository[domain.RegistryItem]) *RegistryHandler {
        return &RegistryHandler{repo: repo}
}

// List — GET /api/v1/registry (доступно всем ролям).
func (h *RegistryHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/registry/{id} (доступно всем ролям).
func (h *RegistryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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

// Create — POST /api/v1/registry (доступно всем ролям: web И android).
func (h *RegistryHandler) Create(w http.ResponseWriter, r *http.Request) {
        var entity domain.RegistryItem
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация: артикул обязателен
        if entity.Article == "" {
                errors.WriteError(w, errors.Validation("артикул обязателен", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/registry/{id} (доступно всем ролям).
func (h *RegistryHandler) Update(w http.ResponseWriter, r *http.Request) {
        id := getURLParam(r, "id")
        if id == "" {
                errors.WriteError(w, errors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.RegistryItem
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация: артикул обязателен
        if entity.Article == "" {
                errors.WriteError(w, errors.Validation("артикул обязателен", nil))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/registry/{id} (доступно всем ролям).
func (h *RegistryHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// RegisterRoutes регистрирует роуты для реестра.
func (h *RegistryHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/registry", h.List)
        mux.HandleFunc("GET /api/v1/registry/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/registry", h.Create)
        mux.HandleFunc("PUT /api/v1/registry/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/registry/{id}", h.Delete)
}