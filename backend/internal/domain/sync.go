package domain

import "encoding/json"

// SyncChange — единичное изменение в дельта-синхронизации (ADR-004).
// Версия — глобальный монотонный счётчик сервера, НЕ timestamp.
type SyncChange struct {
	Module  string          `json:"module"`
	ID      string          `json:"id"`
	Data    json.RawMessage `json:"data,omitempty"`
	Version int64           `json:"version"`
	Deleted bool            `json:"deleted"`
}

// SyncResponse — ответ GET /api/v1/sync?since=N:
// все изменения с version > since, включая tombstones.
type SyncResponse struct {
	Since   int64        `json:"since"`
	Current int64        `json:"current"`
	Changes []SyncChange `json:"changes"`
}

// SyncRequest — тело POST /api/v1/sync: изменения клиента.
type SyncRequest struct {
	DeviceID string       `json:"device_id"`
	Changes  []SyncChange `json:"changes"`
}

// ResolveLWW разрешает конфликт по правилу Last-Write-Win:
// побеждает изменение с большей монотонной версией.
func ResolveLWW(local, remote SyncChange) SyncChange {
	if remote.Version > local.Version {
		return remote
	}
	return local
}
