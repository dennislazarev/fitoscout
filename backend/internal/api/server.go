package api

import (
	"encoding/json"
	"net/http"
	"time"

	"fitoscout/backend/internal/api/handlers"
	"fitoscout/backend/internal/auth"
	"fitoscout/backend/internal/domain/services"
	"fitoscout/backend/internal/storage"
)

const APIVersion = "v1"

type Server struct {
	appVersion  string
	commit      string
	buildDate   string
	db          *storage.DB
	syncService services.SyncService
}

func NewServer(version, commit, buildDate string, db *storage.DB, syncService services.SyncService) *Server {
	return &Server{
		appVersion:  version,
		commit:      commit,
		buildDate:   buildDate,
		db:          db,
		syncService: syncService,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/auth/whoami", s.handleWhoami)
	mux.HandleFunc("GET /api/v1/healthz", s.handleHealthz)
	// Sync endpoints
	if s.syncService != nil {
		syncHandler := handlers.NewSyncHandler(s.syncService)
		mux.HandleFunc("GET /api/v1/sync", syncHandler.GetDelta)
		mux.HandleFunc("POST /api/v1/sync", syncHandler.ApplyChanges)
	}
	return mux
}

// HandleHealthz — публичный хэндлер healthcheck (без mTLS).
// Экспортируется для использования в app.go.
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":         "ok",
		"server_time_ms": time.Now().UnixMilli(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	HandleHealthz(w, r)
}

func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFromContext(r.Context())
	cn := auth.ExtractCN(r)

	certNotAfterMs := int64(0)
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		certNotAfterMs = r.TLS.PeerCertificates[0].NotAfter.UnixMilli()
	}

	response := map[string]any{
		"role":              string(role),
		"cn":                cn,
		"server_time_ms":    time.Now().UnixMilli(),
		"cert_not_after_ms": certNotAfterMs,
		"api_version":       APIVersion,
		"app_version":       s.appVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
