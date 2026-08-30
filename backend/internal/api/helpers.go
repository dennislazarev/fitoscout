package api

import (
	"encoding/json"
	"net/http"

	"fitoscout/backend/internal/errors"
)

// writeError записывает ошибку в формате JSON
func writeError(w http.ResponseWriter, err *errors.AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())
	json.NewEncoder(w).Encode(err.Envelope())
}