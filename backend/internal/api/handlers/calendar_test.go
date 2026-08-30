package handlers_test

import (
        "bytes"
        "context"
        "database/sql"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
        "time"

        "fitoscout/backend/internal/api/handlers"
        "fitoscout/backend/internal/domain"
)

// MockCalendarRepository — мок-репозиторий для тестирования календаря
type MockCalendarRepository struct {
        items map[string]domain.CalendarEvent
        count int64
}

func NewMockCalendarRepository() *MockCalendarRepository {
        return &MockCalendarRepository{items: make(map[string]domain.CalendarEvent)}
}

func (m *MockCalendarRepository) List(ctx context.Context, limit, offset int) ([]domain.CalendarEvent, error) {
        result := make([]domain.CalendarEvent, 0, len(m.items))
        for _, item := range m.items {
                result = append(result, item)
        }
        if len(result) > limit {
                result = result[:limit]
        }
        return result, nil
}

func (m *MockCalendarRepository) GetByID(ctx context.Context, id string) (*domain.CalendarEvent, error) {
        item, ok := m.items[id]
        if !ok {
                return nil, sql.ErrNoRows
        }
        return &item, nil
}

func (m *MockCalendarRepository) Create(ctx context.Context, entity *domain.CalendarEvent) error {
        m.items[entity.ID] = *entity
        m.count++
        return nil
}

func (m *MockCalendarRepository) Update(ctx context.Context, entity *domain.CalendarEvent) error {
        m.items[entity.ID] = *entity
        return nil
}

func (m *MockCalendarRepository) SoftDelete(ctx context.Context, id string) error {
        delete(m.items, id)
        if m.count > 0 {
                m.count--
        }
        return nil
}

func (m *MockCalendarRepository) Count(ctx context.Context) (int64, error) {
        return m.count, nil
}

func (m *MockCalendarRepository) GetSince(ctx context.Context, since int64) ([]domain.CalendarEvent, error) {
        result := make([]domain.CalendarEvent, 0)
        for _, item := range m.items {
                if item.Version > since {
                        result = append(result, item)
                }
        }
        return result, nil
}

func TestCalendarHandler_Create(t *testing.T) {
        t.Run("успешное создание события", func(t *testing.T) {
                repo := NewMockCalendarRepository()
                handler := handlers.NewCalendarHandler(repo)

                event := domain.CalendarEvent{
                        Title:       "Тестовое событие",
                        EventAt:     time.Now().Add(24 * time.Hour),
                        BeforeEvent: 30,
                        RepeatAfter: 7,
                }

                body, _ := json.Marshal(event)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusCreated {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
                }

                var result domain.CalendarEvent
                json.Unmarshal(w.Body.Bytes(), &result)
                if result.Title != event.Title {
                        t.Errorf("ожидалось название %q, получено %q", event.Title, result.Title)
                }
        })

        t.Run("валидация: пустое название", func(t *testing.T) {
                repo := NewMockCalendarRepository()
                handler := handlers.NewCalendarHandler(repo)

                event := domain.CalendarEvent{
                        Title:   "",
                        EventAt: time.Now(),
                }

                body, _ := json.Marshal(event)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })

        t.Run("валидация: дата не указана", func(t *testing.T) {
                repo := NewMockCalendarRepository()
                handler := handlers.NewCalendarHandler(repo)

                event := domain.CalendarEvent{
                        Title:   "Событие без даты",
                        EventAt: time.Time{},
                }

                body, _ := json.Marshal(event)
                req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar", bytes.NewReader(body))
                w := httptest.NewRecorder()

                handler.Create(w, req)

                if w.Code != http.StatusUnprocessableEntity {
                        t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
                }
        })
}

func TestCalendarHandler_List(t *testing.T) {
        repo := NewMockCalendarRepository()
        handler := handlers.NewCalendarHandler(repo)

        // Добавим тестовые данные
        now := time.Now()
        event1 := domain.CalendarEvent{Title: "Событие 1", EventAt: now}
        event1.SetID("id1")
        event2 := domain.CalendarEvent{Title: "Событие 2", EventAt: now}
        event2.SetID("id2")
        repo.items["id1"] = event1
        repo.items["id2"] = event2
        repo.count = 2

        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar", nil)
        w := httptest.NewRecorder()

        handler.List(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
        }

        var result map[string]interface{}
        json.Unmarshal(w.Body.Bytes(), &result)
        data := result["data"].([]interface{})
        if len(data) != 2 {
                t.Errorf("ожидалось 2 события, получено %d", len(data))
        }
}

func TestCalendarHandler_Update(t *testing.T) {
        repo := NewMockCalendarRepository()
        handler := handlers.NewCalendarHandler(repo)

        now := time.Now()
        event := domain.CalendarEvent{
                Title: "Исходное название",
                EventAt: now,
        }
        event.SetID("test-id")
        repo.items["test-id"] = event
        repo.count = 1

        updated := domain.CalendarEvent{
                Title:   "Обновлённое название",
                EventAt: now.Add(time.Hour),
        }

        body, _ := json.Marshal(updated)
        req := httptest.NewRequest(http.MethodPut, "/api/v1/calendar/test-id", bytes.NewReader(body))
        w := httptest.NewRecorder()

        handler.Update(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
        }

        var result domain.CalendarEvent
        json.Unmarshal(w.Body.Bytes(), &result)
        if result.Title != "Обновлённое название" {
                t.Errorf("ожидалось обновлённое название, получено %q", result.Title)
        }
}

func TestCalendarHandler_Delete(t *testing.T) {
        repo := NewMockCalendarRepository()
        handler := handlers.NewCalendarHandler(repo)

        event := domain.CalendarEvent{Title: "Удаляемое событие", EventAt: time.Now()}
        event.SetID("test-id")
        repo.items["test-id"] = event
        repo.count = 1

        req := httptest.NewRequest(http.MethodDelete, "/api/v1/calendar/test-id", nil)
        w := httptest.NewRecorder()

        handler.Delete(w, req)

        if w.Code != http.StatusNoContent {
                t.Errorf("ожидался статус %d, получен %d", http.StatusNoContent, w.Code)
        }

        if _, exists := repo.items["test-id"]; exists {
                t.Error("ожидалось удаление события из репозитория")
        }
}

func TestCalendarHandler_MarkDone(t *testing.T) {
        repo := NewMockCalendarRepository()
        handler := handlers.NewCalendarHandler(repo)

        now := time.Now()
        event := domain.CalendarEvent{
                Title: "Задача",
                EventAt: now,
                DoneAt: nil,
        }
        event.SetID("test-id")
        repo.items["test-id"] = event
        repo.count = 1

        req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar/test-id/done", nil)
        w := httptest.NewRecorder()

        handler.MarkDone(w, req)

        if w.Code != http.StatusOK {
                t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
        }

        var result domain.CalendarEvent
        json.Unmarshal(w.Body.Bytes(), &result)
        if result.DoneAt == nil {
                t.Error("ожидалось, что DoneAt будет установлен")
        }
}