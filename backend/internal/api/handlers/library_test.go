package handlers_test

import (
        "bytes"
        "context"
        "database/sql"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"

        "fitoscout/backend/internal/api/handlers"
        "fitoscout/backend/internal/domain"
)

// MockLibraryRepository — мок-репозиторий для тестирования библиотеки
type MockLibraryRepository struct {
        items map[string]domain.LibraryItem
        count int64
}

func NewMockLibraryRepository() *MockLibraryRepository {
        return &MockLibraryRepository{items: make(map[string]domain.LibraryItem)}
}

func (m *MockLibraryRepository) List(ctx context.Context, limit, offset int) ([]domain.LibraryItem, error) {
        result := make([]domain.LibraryItem, 0, len(m.items))
        for _, item := range m.items {
                result = append(result, item)
        }
        if len(result) > limit {
                result = result[:limit]
        }
        return result, nil
}

func (m *MockLibraryRepository) GetByID(ctx context.Context, id string) (*domain.LibraryItem, error) {
        item, ok := m.items[id]
        if !ok {
                return nil, sql.ErrNoRows
        }
        return &item, nil
}

func (m *MockLibraryRepository) Create(ctx context.Context, entity *domain.LibraryItem) error {
        m.items[entity.ID] = *entity
        m.count++
        return nil
}

func (m *MockLibraryRepository) Update(ctx context.Context, entity *domain.LibraryItem) error {
        m.items[entity.ID] = *entity
        return nil
}

func (m *MockLibraryRepository) SoftDelete(ctx context.Context, id string) error {
        delete(m.items, id)
        if m.count > 0 {
                m.count--
        }
        return nil
}

func (m *MockLibraryRepository) Count(ctx context.Context) (int64, error) {
        return m.count, nil
}

func (m *MockLibraryRepository) GetSince(ctx context.Context, since int64) ([]domain.LibraryItem, error) {
        result := make([]domain.LibraryItem, 0)
        for _, item := range m.items {
                if item.Version > since {
                        result = append(result, item)
                }
        }
        return result, nil
}

func TestLibraryHandler_Create(t *testing.T) {
        t.Run("успешное создание (web)", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Тестовая книга",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/test.pdf",
                }

                body, _ := json.Marshal(item)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/library", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusCreated {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
                }

                var result domain.LibraryItem
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Title != "Тестовая книга" {
                        t.Errorf("ожидался заголовок 'Тестовая книга', получен '%s'", result.Title)
                }
        })

        t.Run("успешное создание (android)", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 2,
                        Title:      "Видео для android",
                        Author:     func() *string { s := "Режиссёр"; return &s }(),
                        Format:     "mp4",
                        FilePath:   "/volume2/Библиотека/video.mp4",
                }

                body, _ := json.Marshal(item)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/library", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusCreated {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
                }
        })

        t.Run("валидация: пустой заголовок", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/test.pdf",
                }

                body, _ := json.Marshal(item)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/library", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })

        t.Run("валидация: пустой путь к файлу", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Книга",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "",
                }

                body, _ := json.Marshal(item)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/library", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })

        t.Run("некорректный JSON", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                req := httptest.NewRequest(http.MethodPost, "/api/v1/library", bytes.NewReader([]byte("invalid json")))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })
}

func TestLibraryHandler_Update(t *testing.T) {
        t.Run("успешное обновление", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Исходная книга",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/original.pdf",
                }
                item.SetID("test-id")
                repo.items["test-id"] = item
                repo.count = 1

                updated := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Обновлённая книга",
                        Author:     func() *string { s := "Новый автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/updated.pdf",
                }

                body, _ := json.Marshal(updated)
                req := httptest.NewRequest(http.MethodPut, "/api/v1/library/test-id", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Update(w, req)

                if w.Code != http.StatusOK {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
                }

                var result domain.LibraryItem
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Title != "Обновлённая книга" {
                        t.Errorf("ожидался заголовок 'Обновлённая книга', получен '%s'", result.Title)
                }
        })

        t.Run("пустой ID", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Книга",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/test.pdf",
                }

                body, _ := json.Marshal(item)
                req := httptest.NewRequest(http.MethodPut, "/api/v1/library/", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Update(w, req)

                if w.Code != http.StatusBadRequest {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
                }
        })
}

func TestLibraryHandler_Delete(t *testing.T) {
        t.Run("успешное удаление", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Удаляемая книга",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/test.pdf",
                }
                item.SetID("test-id")
                repo.items["test-id"] = item
                repo.count = 1

                req := httptest.NewRequest(http.MethodDelete, "/api/v1/library/test-id", nil)
                w := httptest.NewRecorder()

                handler.Delete(w, req)

                if w.Code != http.StatusNoContent {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusNoContent, w.Code)
                }

                if _, exists := repo.items["test-id"]; exists {
                        t.Error("ожидалось удаление элемента из репозитория")
                }
        })

        t.Run("пустой ID", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                req := httptest.NewRequest(http.MethodDelete, "/api/v1/library/", nil)
                w := httptest.NewRecorder()

                handler.Delete(w, req)

                if w.Code != http.StatusBadRequest {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
                }
        })
}

func TestLibraryHandler_List(t *testing.T) {
        t.Run("успешный список", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item1 := domain.LibraryItem{CategoryID: 1, Title: "Книга 1", Format: "pdf"}
                item1.SetID("id1")
                item2 := domain.LibraryItem{CategoryID: 2, Title: "Книга 2", Format: "djvu"}
                item2.SetID("id2")
                repo.items["id1"] = item1
                repo.items["id2"] = item2
                repo.count = 2

                req := httptest.NewRequest(http.MethodGet, "/api/v1/library", nil)
                w := httptest.NewRecorder()

                handler.List(w, req)

                if w.Code != http.StatusOK {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
                }

                var result map[string]interface{}
                json.Unmarshal(w.Body.Bytes(), &result)
                data := result["data"].([]interface{})
                if len(data) != 2 {
                        t.Errorf("ожидалось 2 элемента, получено %d", len(data))
                }
        })
}

func TestLibraryHandler_GetByID(t *testing.T) {
        t.Run("успешное получение", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                item := domain.LibraryItem{
                        CategoryID: 1,
                        Title:      "Тестовая книга",
                        Author:     func() *string { s := "Автор"; return &s }(),
                        Format:     "pdf",
                        FilePath:   "/volume2/Библиотека/test.pdf",
                }
                item.SetID("test-id")
                repo.items["test-id"] = item
                repo.count = 1

                req := httptest.NewRequest(http.MethodGet, "/api/v1/library/test-id", nil)
                w := httptest.NewRecorder()

                handler.GetByID(w, req)

                if w.Code != http.StatusOK {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
                }

                var result domain.LibraryItem
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Title != "Тестовая книга" {
                        t.Errorf("ожидался заголовок 'Тестовая книга', получен '%s'", result.Title)
                }
        })

        t.Run("элемент не найден", func(t *testing.T) {
                repo := NewMockLibraryRepository()
                handler := handlers.NewLibraryHandler(repo, "")

                req := httptest.NewRequest(http.MethodGet, "/api/v1/library/non-existent-id", nil)
                w := httptest.NewRecorder()

                handler.GetByID(w, req)

                if w.Code != http.StatusNotFound {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusNotFound, w.Code)
                }
        })
}