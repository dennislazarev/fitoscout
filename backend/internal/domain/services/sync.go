// Package services содержит интерфейсы доменных сервисов.
package services

import (
	"context"
)

// SyncService — ядро синхронизации (ADR-004: монотонные версии + LWW).
type SyncService interface {
	// GetDelta возвращает дельту изменений с версии since.
	GetDelta(ctx context.Context, deviceID string, since int64) (*SyncDelta, error)

	// ApplyChanges применяет изменения от клиента (с LWW).
	ApplyChanges(ctx context.Context, deviceID string, changes []ClientChange) (*ApplyResult, error)
}

// SyncDelta — дельта изменений с сервера.
type SyncDelta struct {
	DeviceID       string           `json:"device_id"`
	SinceVersion   int64            `json:"since_version"`
	CurrentVersion int64            `json:"current_version"`
	Changes        map[string][]any `json:"changes"` // module_key -> []entity
	ServerTime     int64            `json:"server_time_ms"`
}

// ClientChange — одно изменение от клиента.
type ClientChange struct {
	Module  string `json:"module"`
	ID      string `json:"id"`
	Action  string `json:"action"`  // "create", "update", "delete"
	Data    any    `json:"data"`    // JSON entity (для create/update)
	Version int64  `json:"version"` // версия на клиенте
}

// ApplyResult — результат применения изменений клиента.
type ApplyResult struct {
	Accepted []AppliedChange  `json:"accepted"`
	Rejected []RejectedChange `json:"rejected"`
	Delta    *SyncDelta       `json:"delta"`
}

// AppliedChange — принятое изменение.
type AppliedChange struct {
	Module     string `json:"module"`
	ID         string `json:"id"`
	NewVersion int64  `json:"new_version"`
}

// RejectedChange — отвергнутое изменение (с причиной).
type RejectedChange struct {
	Module        string `json:"module"`
	ID            string `json:"id"`
	Reason        string `json:"reason"` // "conflict_higher_version", "not_found", "unknown_module"
	ServerVersion int64  `json:"server_version,omitempty"`
}
