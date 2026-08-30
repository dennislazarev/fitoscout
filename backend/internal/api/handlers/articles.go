package handlers

import (
        "encoding/json"
        "net/http"

        "fitoscout/backend/internal/auth"
        apperrors "fitoscout/backend/internal/errors"
        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
)

// ArticlesHandler обрабатывает CRUD запросы для модуля "Статьи".
type ArticlesHandler struct {
        repo repositories.Repository[domain.Article]
}

// NewArticlesHandler создаёт новый обработчик для статей.
func NewArticlesHandler(repo repositories.Repository[domain.Article]) *ArticlesHandler {
        return &ArticlesHandler{repo: repo}
}

// List — GET /api/v1/articles?limit=N&offset=M
func (h *ArticlesHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/articles/{id}
func (h *ArticlesHandler) GetByID(w http.ResponseWriter, r *http.Request) {
        id := extractID(r, "articles")
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

// Create — POST /api/v1/articles (только web)
func (h *ArticlesHandler) Create(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("создание доступно только админке"))
                return
        }

        var entity domain.Article
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                apperrors.WriteError(w, apperrors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация
        if entity.Name == "" {
                apperrors.WriteError(w, apperrors.Validation("название статьи обязательно", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                apperrors.WriteError(w, err)
                return
        }

        apperrors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/articles/{id} (только web)
func (h *ArticlesHandler) Update(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("обновление доступно только админке"))
                return
        }

        id := extractID(r, "articles")
        if id == "" {
                apperrors.WriteError(w, apperrors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.Article
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

// Delete — DELETE /api/v1/articles/{id} (только web)
func (h *ArticlesHandler) Delete(w http.ResponseWriter, r *http.Request) {
        role := auth.RoleFromContext(r.Context())
        if role != auth.RoleWeb {
                apperrors.WriteError(w, apperrors.Forbidden("удаление доступно только админке"))
                return
        }

        id := extractID(r, "articles")
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

// RegisterRoutes регистрирует роуты для статей.
func (h *ArticlesHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/articles", h.List)
        mux.HandleFunc("GET /api/v1/articles/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/articles", h.Create)
        mux.HandleFunc("PUT /api/v1/articles/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/articles/{id}", h.Delete)
}