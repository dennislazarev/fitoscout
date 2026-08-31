package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"fitoscout/backend/internal/domain/services"
	apperrors "fitoscout/backend/internal/errors"
)

// SyncHandler — HTTP обработчик для endpoints синхронизации.
type SyncHandler struct {
	syncService services.SyncService
}

// NewSyncHandler создаёт новый SyncHandler.
func NewSyncHandler(syncService services.SyncService) *SyncHandler {
	return &SyncHandler{syncService: syncService}
}

// GetDeltaRequest — запрос на получение дельты.
type GetDeltaRequest struct {
	Since int64 `json:"since"`
}

// ApplyChangesRequest — запрос на применение изменений.
type ApplyChangesRequest struct {
	Changes []services.ClientChange `json:"changes"`
}

// GetDelta обрабатывает GET /api/v1/sync?since=N.
// Доступно всем ролям.
func (h *SyncHandler) GetDelta(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем deviceID из контекста (устанавливается middleware аутентификации)
	deviceID, ok := ctx.Value("device_id").(string)
	if !ok || deviceID == "" {
		apperrors.WriteError(w, apperrors.Unauthorized("требуется аутентификация"))
		return
	}

	// Парсим since из query параметров
	sinceStr := r.URL.Query().Get("since")
	var since int64 = 0
	if sinceStr != "" {
		var err error
		since, err = strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			apperrors.WriteError(w, apperrors.BadRequest("неверный формат since"))
			return
		}
	}

	delta, err := h.syncService.GetDelta(ctx, deviceID, since)
	if err != nil {
		apperrors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(delta)
}

// ApplyChanges обрабатывает POST /api/v1/sync.
// Доступно только роли web.
func (h *SyncHandler) ApplyChanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем deviceID из контекста
	deviceID, ok := ctx.Value("device_id").(string)
	if !ok || deviceID == "" {
		apperrors.WriteError(w, apperrors.Unauthorized("требуется аутентификация"))
		return
	}

	// Проверяем роль (только web может применять изменения)
	role, _ := ctx.Value("role").(string)
	if role != "web" {
		apperrors.WriteError(w, apperrors.Forbidden("требуется роль web"))
		return
	}

	var req ApplyChangesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperrors.WriteError(w, apperrors.BadRequest("ошибка парсинга JSON"))
		return
	}

	if len(req.Changes) == 0 {
		apperrors.WriteError(w, apperrors.BadRequest("пустой список изменений"))
		return
	}

	result, err := h.syncService.ApplyChanges(ctx, deviceID, req.Changes)
	if err != nil {
		apperrors.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
