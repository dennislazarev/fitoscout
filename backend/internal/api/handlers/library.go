package handlers

import (
        "encoding/json"
        "net/http"
        "os"
        "path/filepath"
        "strings"

        "fitoscout/backend/internal/domain"
        "fitoscout/backend/internal/domain/repositories"
        "fitoscout/backend/internal/errors"
)

// LibraryHandler обрабатывает HTTP-запросы к модулю "Библиотека".
type LibraryHandler struct {
        repo      repositories.Repository[domain.LibraryItem]
        basePath  string // базовый путь к файлам библиотеки
}

// NewLibraryHandler создаёт новый хэндлер библиотеки.
func NewLibraryHandler(repo repositories.Repository[domain.LibraryItem], basePath string) *LibraryHandler {
        return &LibraryHandler{repo: repo, basePath: basePath}
}

// List — GET /api/v1/library (доступно всем ролям).
func (h *LibraryHandler) List(w http.ResponseWriter, r *http.Request) {
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

// GetByID — GET /api/v1/library/{id} (доступно всем ролям).
func (h *LibraryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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

// Create — POST /api/v1/library (доступно всем ролям: web И android).
func (h *LibraryHandler) Create(w http.ResponseWriter, r *http.Request) {
        var entity domain.LibraryItem
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        // Валидация: название обязательно
        if entity.Title == "" {
                errors.WriteError(w, errors.Validation("название обязательно", nil))
                return
        }

        // Валидация: формат обязателен
        if entity.Format == "" {
                errors.WriteError(w, errors.Validation("формат файла обязателен", nil))
                return
        }

        // Валидация: поддерживаемые форматы
        validFormats := map[string]bool{"pdf": true, "djvu": true, "mp4": true, "mkv": true}
        if !validFormats[entity.Format] {
                errors.WriteError(w, errors.Validation("неподдерживаемый формат файла", nil))
                return
        }

        // Валидация: путь к файлу обязателен
        if entity.FilePath == "" {
                errors.WriteError(w, errors.Validation("путь к файлу обязателен", nil))
                return
        }

        if err := h.repo.Create(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusCreated, entity)
}

// Update — PUT /api/v1/library/{id} (доступно всем ролям).
func (h *LibraryHandler) Update(w http.ResponseWriter, r *http.Request) {
        id := getURLParam(r, "id")
        if id == "" {
                errors.WriteError(w, errors.BadRequest("ID обязателен"))
                return
        }

        var entity domain.LibraryItem
        if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
                errors.WriteError(w, errors.Validation("некорректный JSON", nil))
                return
        }

        entity.SetID(id)

        // Валидация: название обязательно
        if entity.Title == "" {
                errors.WriteError(w, errors.Validation("название обязательно", nil))
                return
        }

        if err := h.repo.Update(r.Context(), &entity); err != nil {
                errors.WriteError(w, err)
                return
        }

        errors.WriteJSON(w, http.StatusOK, entity)
}

// Delete — DELETE /api/v1/library/{id} (доступно всем ролям).
func (h *LibraryHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// Stream — GET /api/v1/library/{id}/stream (HTTP Range support).
func (h *LibraryHandler) Stream(w http.ResponseWriter, r *http.Request) {
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

        // Открыть файл
        filePath := entity.FilePath
        if !filepath.IsAbs(filePath) && h.basePath != "" {
                filePath = filepath.Join(h.basePath, filePath)
        }

        file, err := os.Open(filePath)
        if err != nil {
                errors.WriteError(w, errors.NotFound("файл не найден"))
                return
        }
        defer file.Close()

        // Получить размер файла
        stat, err := file.Stat()
        if err != nil {
                errors.WriteError(w, errors.Internal("ошибка получения информации о файле"))
                return
        }

        // Установить заголовки для streaming
        contentType := h.getContentType(entity.Format)
        w.Header().Set("Content-Type", contentType)
        w.Header().Set("Accept-Ranges", "bytes")
        w.Header().Set("Content-Disposition", `inline; filename="`+filepath.Base(entity.FilePath)+`"`)

        // HTTP Range support через http.ServeContent
        http.ServeContent(w, r, entity.Title, stat.ModTime(), file)
}

// getContentType возвращает MIME-тип по формату файла.
func (h *LibraryHandler) getContentType(format string) string {
        switch strings.ToLower(format) {
        case "pdf":
                return "application/pdf"
        case "djvu":
                return "image/vnd.djvu"
        case "mp4":
                return "video/mp4"
        case "mkv":
                return "video/x-matroska"
        default:
                return "application/octet-stream"
        }
}

// RegisterRoutes регистрирует роуты для библиотеки.
func (h *LibraryHandler) RegisterRoutes(mux *http.ServeMux) {
        mux.HandleFunc("GET /api/v1/library", h.List)
        mux.HandleFunc("GET /api/v1/library/{id}", h.GetByID)
        mux.HandleFunc("POST /api/v1/library", h.Create)
        mux.HandleFunc("PUT /api/v1/library/{id}", h.Update)
        mux.HandleFunc("DELETE /api/v1/library/{id}", h.Delete)
        mux.HandleFunc("GET /api/v1/library/{id}/stream", h.Stream)
}