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

// MockRegistryRepository — мок-репозиторий для тестирования реестра
type MockRegistryRepository struct {
items map[string]domain.RegistryItem
count int64
}

func NewMockRegistryRepository() *MockRegistryRepository {
return &MockRegistryRepository{items: make(map[string]domain.RegistryItem)}
}

func (m *MockRegistryRepository) List(ctx context.Context, limit, offset int) ([]domain.RegistryItem, error) {
result := make([]domain.RegistryItem, 0, len(m.items))
for _, item := range m.items {
result = append(result, item)
}
if len(result) > limit {
result = result[:limit]
}
return result, nil
}

func (m *MockRegistryRepository) GetByID(ctx context.Context, id string) (*domain.RegistryItem, error) {
item, ok := m.items[id]
if !ok {
return nil, sql.ErrNoRows
}
return &item, nil
}

func (m *MockRegistryRepository) Create(ctx context.Context, entity *domain.RegistryItem) error {
m.items[entity.ID] = *entity
m.count++
return nil
}

func (m *MockRegistryRepository) Update(ctx context.Context, entity *domain.RegistryItem) error {
m.items[entity.ID] = *entity
return nil
}

func (m *MockRegistryRepository) SoftDelete(ctx context.Context, id string) error {
delete(m.items, id)
if m.count > 0 {
m.count--
}
return nil
}

func (m *MockRegistryRepository) Count(ctx context.Context) (int64, error) {
return m.count, nil
}

func (m *MockRegistryRepository) GetSince(ctx context.Context, since int64) ([]domain.RegistryItem, error) {
result := make([]domain.RegistryItem, 0)
for _, item := range m.items {
if item.Version > since {
result = append(result, item)
}
}
return result, nil
}

func strPtr(s string) *string { return &s }

func TestRegistryHandler_Create(t *testing.T) {
t.Run("успешное создание (web)", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "ART-001",
IsLost:  false,
PlantID: strPtr("plant-id-123"),
}

body, _ := json.Marshal(entry)
req := httptest.NewRequest(http.MethodPost, "/api/v1/registry", bytes.NewReader(body))
w := httptest.NewRecorder()

handler.Create(w, req)

if w.Code != http.StatusCreated {
t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
}

var result domain.RegistryItem
json.Unmarshal(w.Body.Bytes(), &result)
if result.Article != "ART-001" {
t.Errorf("ожидался артикул 'ART-001', получен '%s'", result.Article)
}
})

t.Run("успешное создание (android)", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "ART-002",
IsLost:  true,
PlantID: strPtr("plant-id-456"),
}

body, _ := json.Marshal(entry)
req := httptest.NewRequest(http.MethodPost, "/api/v1/registry", bytes.NewReader(body))
w := httptest.NewRecorder()

handler.Create(w, req)

if w.Code != http.StatusCreated {
t.Errorf("ожидался статус %d, получен %d", http.StatusCreated, w.Code)
}
})

t.Run("валидация: пустой артикул", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "",
IsLost:  false,
PlantID: strPtr("plant-id-123"),
}

body, _ := json.Marshal(entry)
req := httptest.NewRequest(http.MethodPost, "/api/v1/registry", bytes.NewReader(body))
w := httptest.NewRecorder()

handler.Create(w, req)

if w.Code != http.StatusUnprocessableEntity {
t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
}
})

t.Run("некорректный JSON", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

req := httptest.NewRequest(http.MethodPost, "/api/v1/registry", bytes.NewReader([]byte("invalid json")))
w := httptest.NewRecorder()

handler.Create(w, req)

if w.Code != http.StatusUnprocessableEntity {
t.Errorf("ожидался статус %d, получен %d", http.StatusUnprocessableEntity, w.Code)
}
})
}

func TestRegistryHandler_Update(t *testing.T) {
t.Run("успешное обновление", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "ART-001",
IsLost:  false,
PlantID: strPtr("plant-id-123"),
}
entry.SetID("test-id")
repo.items["test-id"] = entry
repo.count = 1

updated := domain.RegistryItem{
Article: "ART-001-UPDATED",
IsLost:  true,
PlantID: strPtr("plant-id-789"),
}

body, _ := json.Marshal(updated)
req := httptest.NewRequest(http.MethodPut, "/api/v1/registry/test-id", bytes.NewReader(body))
w := httptest.NewRecorder()

handler.Update(w, req)

if w.Code != http.StatusOK {
t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
}

var result domain.RegistryItem
json.Unmarshal(w.Body.Bytes(), &result)
if result.Article != "ART-001-UPDATED" {
t.Errorf("ожидался артикул 'ART-001-UPDATED', получен '%s'", result.Article)
}
})

t.Run("пустой ID", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "ART-001",
IsLost:  false,
PlantID: strPtr("plant-id-123"),
}

body, _ := json.Marshal(entry)
req := httptest.NewRequest(http.MethodPut, "/api/v1/registry/", bytes.NewReader(body))
w := httptest.NewRecorder()

handler.Update(w, req)

if w.Code != http.StatusBadRequest {
t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
}
})
}

func TestRegistryHandler_Delete(t *testing.T) {
t.Run("успешное удаление", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "ART-001",
IsLost:  false,
PlantID: strPtr("plant-id-123"),
}
entry.SetID("test-id")
repo.items["test-id"] = entry
repo.count = 1

req := httptest.NewRequest(http.MethodDelete, "/api/v1/registry/test-id", nil)
w := httptest.NewRecorder()

handler.Delete(w, req)

if w.Code != http.StatusNoContent {
t.Errorf("ожидался статус %d, получен %d", http.StatusNoContent, w.Code)
}

if _, exists := repo.items["test-id"]; exists {
t.Error("ожидалось удаление записи из репозитория")
}
})

t.Run("пустой ID", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

req := httptest.NewRequest(http.MethodDelete, "/api/v1/registry/", nil)
w := httptest.NewRecorder()

handler.Delete(w, req)

if w.Code != http.StatusBadRequest {
t.Errorf("ожидался статус %d, получен %d", http.StatusBadRequest, w.Code)
}
})
}

func TestRegistryHandler_List(t *testing.T) {
t.Run("успешный список", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry1 := domain.RegistryItem{Article: "ART-001", IsLost: false}
entry1.SetID("id1")
entry2 := domain.RegistryItem{Article: "ART-002", IsLost: true}
entry2.SetID("id2")
repo.items["id1"] = entry1
repo.items["id2"] = entry2
repo.count = 2

req := httptest.NewRequest(http.MethodGet, "/api/v1/registry", nil)
w := httptest.NewRecorder()

handler.List(w, req)

if w.Code != http.StatusOK {
t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
}

var result map[string]interface{}
json.Unmarshal(w.Body.Bytes(), &result)
data := result["data"].([]interface{})
if len(data) != 2 {
t.Errorf("ожидалось 2 записи, получено %d", len(data))
}
})
}

func TestRegistryHandler_GetByID(t *testing.T) {
t.Run("успешное получение", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

entry := domain.RegistryItem{
Article: "ART-001",
IsLost:  false,
PlantID: strPtr("plant-id-123"),
}
entry.SetID("test-id")
repo.items["test-id"] = entry
repo.count = 1

req := httptest.NewRequest(http.MethodGet, "/api/v1/registry/test-id", nil)
w := httptest.NewRecorder()

handler.GetByID(w, req)

if w.Code != http.StatusOK {
t.Errorf("ожидался статус %d, получен %d", http.StatusOK, w.Code)
}

var result domain.RegistryItem
json.Unmarshal(w.Body.Bytes(), &result)
if result.Article != "ART-001" {
t.Errorf("ожидался артикул 'ART-001', получен '%s'", result.Article)
}
})

t.Run("запись не найдена", func(t *testing.T) {
repo := NewMockRegistryRepository()
handler := handlers.NewRegistryHandler(repo)

req := httptest.NewRequest(http.MethodGet, "/api/v1/registry/non-existent-id", nil)
w := httptest.NewRecorder()

handler.GetByID(w, req)

if w.Code != http.StatusNotFound {
t.Errorf("ожидался статус %d, получен %d", http.StatusNotFound, w.Code)
}
})
}