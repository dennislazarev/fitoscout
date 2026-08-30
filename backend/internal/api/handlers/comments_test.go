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
// MockCommentsRepository — мок-репозиторий для тестирования комментариев
type MockCommentsRepository struct {
        items map[string]domain.Comment
        count int64
}
func NewMockCommentsRepository() *MockCommentsRepository {
        return &MockCommentsRepository{items: make(map[string]domain.Comment)}
}
func (m *MockCommentsRepository) List(ctx context.Context, limit, offset int) ([]domain.Comment, error) {
        result := make([]domain.Comment, 0, len(m.items))
        for _, item := range m.items {
                result = append(result, item)
        }
        if len(result) > limit {
                result = result[:limit]
        }
        return result, nil
}
func (m *MockCommentsRepository) GetByID(ctx context.Context, id string) (*domain.Comment, error) {
        item, ok := m.items[id]
        if !ok {
                return nil, sql.ErrNoRows
        }
        return &item, nil
}
func (m *MockCommentsRepository) Create(ctx context.Context, entity *domain.Comment) error {
        m.items[entity.ID] = *entity
        m.count++
        return nil
}
func (m *MockCommentsRepository) Update(ctx context.Context, entity *domain.Comment) error {
        m.items[entity.ID] = *entity
        return nil
}
func (m *MockCommentsRepository) SoftDelete(ctx context.Context, id string) error {
        delete(m.items, id)
        if m.count > 0 {
                m.count--
        }
        return nil
}
func (m *MockCommentsRepository) Count(ctx context.Context) (int64, error) {
        return m.count, nil
}
func (m *MockCommentsRepository) GetSince(ctx context.Context, since int64) ([]domain.Comment, error) {
        result := make([]domain.Comment, 0)
        for _, item := range m.items {
                if item.Version > since {
                        result = append(result, item)
                }
        }
        return result, nil
}
func TestCommentsHandler_Create(t *testing.T) {
        t.Run("успешное создание (web)", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Тестовый комментарий",
                        Status:    "new",
                }
                body, _ := json.Marshal(comment)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Create(w, req)
                if w.Code != http.StatusCreated {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
                }
                var result domain.Comment
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Text != "Тестовый комментарий" {
                        t.Errorf("ожидался текст 'Тестовый комментарий', получен '%s'", result.Text)
                }
        })
        t.Run("успешное создание (android)", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "diseases",
                        EntityID:  "test-entity-id",
                        Type:      "task",
                        Text:      "Задача для android",
                        Status:    "new",
                }
                body, _ := json.Marshal(comment)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Create(w, req)
                if w.Code != http.StatusCreated {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
                }
        })
        t.Run("валидация: пустой текст", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "",
                        Status:    "new",
                }
                body, _ := json.Marshal(comment)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Create(w, req)
                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })
        t.Run("валидация: пустой module_key", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Тест",
                        Status:    "new",
                }
                body, _ := json.Marshal(comment)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Create(w, req)
                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })
        t.Run("валидация: пустой entity_id", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "",
                        Type:      "comment",
                        Text:      "Тест",
                        Status:    "new",
                }
                body, _ := json.Marshal(comment)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Create(w, req)
                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })
        t.Run("некорректный JSON", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader([]byte("invalid json")))
                w := httptest.NewRecorder()
                handler.Create(w, req)
                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })
}
func TestCommentsHandler_Update(t *testing.T) {
        t.Run("успешное обновление", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Исходный комментарий",
                        Status:    "new",
                }
                comment.SetID("test-id")
                repo.items["test-id"] = comment
                repo.count = 1
                updated := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Обновлённый комментарий",
                        Status:    "in_progress",
                }
                body, _ := json.Marshal(updated)
                req := httptest.NewRequest(http.MethodPut, "/api/v1/comments/test-id", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Update(w, req)
                if w.Code != http.StatusOK {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
                }
                var result domain.Comment
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Text != "Обновлённый комментарий" {
                        t.Errorf("ожидался текст 'Обновлённый комментарий', получен '%s'", result.Text)
                }
        })
        t.Run("пустой ID", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Тест",
                        Status:    "new",
                }
                body, _ := json.Marshal(comment)
                req := httptest.NewRequest(http.MethodPut, "/api/v1/comments/", bytes.NewReader(body))
                w := httptest.NewRecorder()
                handler.Update(w, req)
                if w.Code != http.StatusBadRequest {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
                }
        })
}
func TestCommentsHandler_Delete(t *testing.T) {
        t.Run("успешное удаление", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Удаляемый комментарий",
                        Status:    "new",
                }
                comment.SetID("test-id")
                repo.items["test-id"] = comment
                repo.count = 1
                req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/test-id", nil)
                w := httptest.NewRecorder()
                handler.Delete(w, req)
                if w.Code != http.StatusNoContent {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusNoContent, w.Code)
                }
                if _, exists := repo.items["test-id"]; exists {
                        t.Error("ожидалось удаление комментария из репозитория")
                }
        })
        t.Run("пустой ID", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/", nil)
                w := httptest.NewRecorder()
                handler.Delete(w, req)
                if w.Code != http.StatusBadRequest {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
                }
        })
}
func TestCommentsHandler_List(t *testing.T) {
        t.Run("успешный список", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment1 := domain.Comment{ModuleKey: "plants", EntityID: "id1", Text: "Комментарий 1"}
                comment1.SetID("id1")
                comment2 := domain.Comment{ModuleKey: "diseases", EntityID: "id2", Text: "Комментарий 2"}
                comment2.SetID("id2")
                repo.items["id1"] = comment1
                repo.items["id2"] = comment2
                repo.count = 2
                req := httptest.NewRequest(http.MethodGet, "/api/v1/comments", nil)
                w := httptest.NewRecorder()
                handler.List(w, req)
                if w.Code != http.StatusOK {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
                }
                var result map[string]interface{}
                json.Unmarshal(w.Body.Bytes(), &result)
                data := result["data"].([]interface{})
                if len(data) != 2 {
                        t.Errorf("ожидалось 2 комментария, получено %d", len(data))
                }
        })
}
func TestCommentsHandler_GetByID(t *testing.T) {
        t.Run("успешное получение", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                comment := domain.Comment{
                        ModuleKey: "plants",
                        EntityID:  "test-entity-id",
                        Type:      "comment",
                        Text:      "Тестовый комментарий",
                        Status:    "new",
                }
                comment.SetID("test-id")
                repo.items["test-id"] = comment
                repo.count = 1
                req := httptest.NewRequest(http.MethodGet, "/api/v1/comments/test-id", nil)
                w := httptest.NewRecorder()
                handler.GetByID(w, req)
                if w.Code != http.StatusOK {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
                }
                var result domain.Comment
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Text != "Тестовый комментарий" {
                        t.Errorf("ожидался текст 'Тестовый комментарий', получен '%s'", result.Text)
                }
        })
        t.Run("комментарий не найден", func(t *testing.T) {
                repo := NewMockCommentsRepository()
                handler := handlers.NewCommentsHandler(repo)
                req := httptest.NewRequest(http.MethodGet, "/api/v1/comments/non-existent-id", nil)
                w := httptest.NewRecorder()
                handler.GetByID(w, req)
                if w.Code != http.StatusNotFound {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusNotFound, w.Code)
                }
        })
}