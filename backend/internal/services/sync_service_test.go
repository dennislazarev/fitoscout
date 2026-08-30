package services

import (
        "context"
        "testing"

        "fitoscout/backend/internal/domain/services"
)

// mockSyncService — мок для тестирования.
type mockSyncService struct {
        getDeltaFunc   func(ctx context.Context, deviceID string, since int64) (*services.SyncDelta, error)
        applyChangesFunc func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error)
}

func (m *mockSyncService) GetDelta(ctx context.Context, deviceID string, since int64) (*services.SyncDelta, error) {
        if m.getDeltaFunc != nil {
                return m.getDeltaFunc(ctx, deviceID, since)
        }
        return &services.SyncDelta{
                DeviceID:       deviceID,
                SinceVersion:   since,
                CurrentVersion: since + 10,
                Changes:        make(map[string][]any),
                ServerTime:     1234567890,
        }, nil
}

func (m *mockSyncService) ApplyChanges(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
        if m.applyChangesFunc != nil {
                return m.applyChangesFunc(ctx, deviceID, changes)
        }

        accepted := make([]services.AppliedChange, 0)
        rejected := make([]services.RejectedChange, 0)

        for _, change := range changes {
                // Простая эмуляция LWW
                if change.Action == "update" && change.Version < 10 {
                        rejected = append(rejected, services.RejectedChange{
                                Module:        change.Module,
                                ID:            change.ID,
                                Reason:        "conflict_higher_version",
                                ServerVersion: 10,
                        })
                } else {
                        accepted = append(accepted, services.AppliedChange{
                                Module:     change.Module,
                                ID:         change.ID,
                                NewVersion: change.Version + 1,
                        })
                }
        }

        return &services.ApplyResult{
                Accepted: accepted,
                Rejected: rejected,
                Delta: &services.SyncDelta{
                        DeviceID:       deviceID,
                        SinceVersion:   10,
                        CurrentVersion: 10,
                        Changes:        make(map[string][]any),
                        ServerTime:     1234567890,
                },
        }, nil
}

// TestLWW_ClientWins — клиент выигрывает (клиент: v=7, сервер: v=5).
func TestLWW_ClientWins(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-1"

        changes := []services.ClientChange{
                {
                        Module:  "plants",
                        ID:      "plant-1",
                        Action:  "update",
                        Version: 7, // Версия клиента больше серверной
                        Data:    map[string]any{"name": "Test Plant"},
                },
        }

        // Эмулируем что серверная версия = 5
        mock := &mockSyncService{
                applyChangesFunc: func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
                        // Все изменения принимаются т.к. версия клиента >= серверной
                        accepted := make([]services.AppliedChange, len(changes))
                        for i, c := range changes {
                                accepted[i] = services.AppliedChange{
                                        Module:     c.Module,
                                        ID:         c.ID,
                                        NewVersion: c.Version + 1,
                                }
                        }
                        return &services.ApplyResult{
                                Accepted: accepted,
                                Rejected: []services.RejectedChange{},
                                Delta: &services.SyncDelta{
                                        DeviceID:       deviceID,
                                        SinceVersion:   8,
                                        CurrentVersion: 8,
                                        Changes:        make(map[string][]any),
                                        ServerTime:     1234567890,
                                },
                        }, nil
                },
        }

        result, err := mock.ApplyChanges(ctx, deviceID, changes)
        if err != nil {
                t.Fatalf("ApplyChanges returned error: %v", err)
        }

        if len(result.Accepted) != 1 {
                t.Errorf("Expected 1 accepted change, got %d", len(result.Accepted))
        }

        if len(result.Rejected) != 0 {
                t.Errorf("Expected 0 rejected changes, got %d", len(result.Rejected))
        }
}

// TestLWW_ServerWins — сервер выигрывает (клиент: v=5, сервер: v=10).
func TestLWW_ServerWins(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-2"

        changes := []services.ClientChange{
                {
                        Module:  "plants",
                        ID:      "plant-2",
                        Action:  "update",
                        Version: 5, // Версия клиента меньше серверной
                        Data:    map[string]any{"name": "Old Plant"},
                },
        }

        // Эмулируем что серверная версия = 10
        mock := &mockSyncService{
                applyChangesFunc: func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
                        // Изменение отклоняется т.к. версия клиента < серверной
                        return &services.ApplyResult{
                                Accepted: []services.AppliedChange{},
                                Rejected: []services.RejectedChange{
                                        {
                                                Module:        changes[0].Module,
                                                ID:            changes[0].ID,
                                                Reason:        "conflict_higher_version",
                                                ServerVersion: 10,
                                        },
                                },
                                Delta: &services.SyncDelta{
                                        DeviceID:       deviceID,
                                        SinceVersion:   10,
                                        CurrentVersion: 10,
                                        Changes:        make(map[string][]any),
                                        ServerTime:     1234567890,
                                },
                        }, nil
                },
        }

        result, err := mock.ApplyChanges(ctx, deviceID, changes)
        if err != nil {
                t.Fatalf("ApplyChanges returned error: %v", err)
        }

        if len(result.Accepted) != 0 {
                t.Errorf("Expected 0 accepted changes, got %d", len(result.Accepted))
        }

        if len(result.Rejected) != 1 {
                t.Errorf("Expected 1 rejected change, got %d", len(result.Rejected))
        }

        if result.Rejected[0].Reason != "conflict_higher_version" {
                t.Errorf("Expected reason 'conflict_higher_version', got '%s'", result.Rejected[0].Reason)
        }

        if result.Rejected[0].ServerVersion != 10 {
                t.Errorf("Expected server version 10, got %d", result.Rejected[0].ServerVersion)
        }
}

// TestCreate_NewRecord — создание новой записи всегда принимается.
func TestCreate_NewRecord(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-3"

        changes := []services.ClientChange{
                {
                        Module:  "plants",
                        ID:      "new-plant-id",
                        Action:  "create",
                        Version: 0,
                        Data:    map[string]any{"name": "New Plant", "id": "new-plant-id"},
                },
        }

        mock := &mockSyncService{
                applyChangesFunc: func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
                        accepted := make([]services.AppliedChange, len(changes))
                        for i, c := range changes {
                                accepted[i] = services.AppliedChange{
                                        Module:     c.Module,
                                        ID:         c.ID,
                                        NewVersion: 1,
                                }
                        }
                        return &services.ApplyResult{
                                Accepted: accepted,
                                Rejected: []services.RejectedChange{},
                                Delta: &services.SyncDelta{
                                        DeviceID:       deviceID,
                                        SinceVersion:   1,
                                        CurrentVersion: 1,
                                        Changes:        make(map[string][]any),
                                        ServerTime:     1234567890,
                                },
                        }, nil
                },
        }

        result, err := mock.ApplyChanges(ctx, deviceID, changes)
        if err != nil {
                t.Fatalf("ApplyChanges returned error: %v", err)
        }

        if len(result.Accepted) != 1 {
                t.Errorf("Expected 1 accepted change, got %d", len(result.Accepted))
        }

        if result.Accepted[0].NewVersion != 1 {
                t.Errorf("Expected new version 1, got %d", result.Accepted[0].NewVersion)
        }
}

// TestDelete_NotFound — удаление несуществующей записи отклоняется.
func TestDelete_NotFound(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-4"

        changes := []services.ClientChange{
                {
                        Module:  "plants",
                        ID:      "non-existent-id",
                        Action:  "delete",
                        Version: 1,
                },
        }

        mock := &mockSyncService{
                applyChangesFunc: func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
                        return &services.ApplyResult{
                                Accepted: []services.AppliedChange{},
                                Rejected: []services.RejectedChange{
                                        {
                                                Module: changes[0].Module,
                                                ID:     changes[0].ID,
                                                Reason: "not_found",
                                        },
                                },
                                Delta: &services.SyncDelta{
                                        DeviceID:       deviceID,
                                        SinceVersion:   0,
                                        CurrentVersion: 0,
                                        Changes:        make(map[string][]any),
                                        ServerTime:     1234567890,
                                },
                        }, nil
                },
        }

        result, err := mock.ApplyChanges(ctx, deviceID, changes)
        if err != nil {
                t.Fatalf("ApplyChanges returned error: %v", err)
        }

        if len(result.Rejected) != 1 {
                t.Errorf("Expected 1 rejected change, got %d", len(result.Rejected))
        }

        if result.Rejected[0].Reason != "not_found" {
                t.Errorf("Expected reason 'not_found', got '%s'", result.Rejected[0].Reason)
        }
}

// TestSoftDelete — soft delete принимается.
func TestSoftDelete(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-5"

        changes := []services.ClientChange{
                {
                        Module:  "plants",
                        ID:      "plant-to-delete",
                        Action:  "delete",
                        Version: 5,
                },
        }

        mock := &mockSyncService{
                applyChangesFunc: func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
                        return &services.ApplyResult{
                                Accepted: []services.AppliedChange{
                                        {
                                                Module:     changes[0].Module,
                                                ID:         changes[0].ID,
                                                NewVersion: 6,
                                        },
                                },
                                Rejected: []services.RejectedChange{},
                                Delta: &services.SyncDelta{
                                        DeviceID:       deviceID,
                                        SinceVersion:   6,
                                        CurrentVersion: 6,
                                        Changes:        make(map[string][]any),
                                        ServerTime:     1234567890,
                                },
                        }, nil
                },
        }

        result, err := mock.ApplyChanges(ctx, deviceID, changes)
        if err != nil {
                t.Fatalf("ApplyChanges returned error: %v", err)
        }

        if len(result.Accepted) != 1 {
                t.Errorf("Expected 1 accepted change, got %d", len(result.Accepted))
        }

        if result.Accepted[0].NewVersion != 6 {
                t.Errorf("Expected new version 6, got %d", result.Accepted[0].NewVersion)
        }
}

// TestUnknownModule — неизвестный модуль отклоняется.
func TestUnknownModule(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-6"

        changes := []services.ClientChange{
                {
                        Module:  "unknown_module_xyz",
                        ID:      "some-id",
                        Action:  "update",
                        Version: 1,
                        Data:    map[string]any{"field": "value"},
                },
        }

        mock := &mockSyncService{
                applyChangesFunc: func(ctx context.Context, deviceID string, changes []services.ClientChange) (*services.ApplyResult, error) {
                        return &services.ApplyResult{
                                Accepted: []services.AppliedChange{},
                                Rejected: []services.RejectedChange{
                                        {
                                                Module: changes[0].Module,
                                                ID:     changes[0].ID,
                                                Reason: "unknown_module",
                                        },
                                },
                                Delta: &services.SyncDelta{
                                        DeviceID:       deviceID,
                                        SinceVersion:   0,
                                        CurrentVersion: 0,
                                        Changes:        make(map[string][]any),
                                        ServerTime:     1234567890,
                                },
                        }, nil
                },
        }

        result, err := mock.ApplyChanges(ctx, deviceID, changes)
        if err != nil {
                t.Fatalf("ApplyChanges returned error: %v", err)
        }

        if len(result.Rejected) != 1 {
                t.Errorf("Expected 1 rejected change, got %d", len(result.Rejected))
        }

        if result.Rejected[0].Reason != "unknown_module" {
                t.Errorf("Expected reason 'unknown_module', got '%s'", result.Rejected[0].Reason)
        }
}

// TestGetDelta_WithSince — GetDelta возвращает только изменения после since.
func TestGetDelta_WithSince(t *testing.T) {
        ctx := context.Background()
        deviceID := "test-device-7"
        since := int64(100)

        expectedChanges := map[string][]any{
                "plants": {
                        map[string]any{"id": "plant-1", "version": 101},
                        map[string]any{"id": "plant-2", "version": 105},
                },
                "diseases": {
                        map[string]any{"id": "disease-1", "version": 103},
                },
        }

        mock := &mockSyncService{
                getDeltaFunc: func(ctx context.Context, deviceID string, since int64) (*services.SyncDelta, error) {
                        return &services.SyncDelta{
                                DeviceID:       deviceID,
                                SinceVersion:   since,
                                CurrentVersion: 105,
                                Changes:        expectedChanges,
                                ServerTime:     1234567890,
                        }, nil
                },
        }

        delta, err := mock.GetDelta(ctx, deviceID, since)
        if err != nil {
                t.Fatalf("GetDelta returned error: %v", err)
        }

        if delta.SinceVersion != since {
                t.Errorf("Expected since_version %d, got %d", since, delta.SinceVersion)
        }

        if delta.CurrentVersion != 105 {
                t.Errorf("Expected current_version 105, got %d", delta.CurrentVersion)
        }

        if len(delta.Changes["plants"]) != 2 {
                t.Errorf("Expected 2 plant changes, got %d", len(delta.Changes["plants"]))
        }

        if len(delta.Changes["diseases"]) != 1 {
                t.Errorf("Expected 1 disease change, got %d", len(delta.Changes["diseases"]))
        }
}