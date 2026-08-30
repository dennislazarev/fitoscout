package errors

import (
        "encoding/json"
        "net/http"
)

// HTTPStatus сопоставляет код ошибки с HTTP-статусом.
func HTTPStatus(code Code) int {
        switch code {
        case CodeUnauthorized:
                return http.StatusUnauthorized
        case CodeForbidden:
                return http.StatusForbidden
        case CodeNotFound:
                return http.StatusNotFound
		case CodeBadRequest:
                return http.StatusBadRequest
        case CodeValidation:
                return http.StatusUnprocessableEntity
        case CodeConflict, CodeDuplicate:
                return http.StatusConflict
        case CodeRateLimited:
                return http.StatusTooManyRequests
        default:
                return http.StatusInternalServerError
        }
}

type errorBody struct {
        Code    Code   `json:"code"`
        Message string `json:"message"`
        Details any    `json:"details,omitempty"`
}

type errorEnvelope struct {
        Error errorBody `json:"error"`
}

// WriteError сериализует ошибку в единый JSON-формат:
// {"error": {"code": "...", "message": "..."}}.
func WriteError(w http.ResponseWriter, err error) {
        appErr := FromError(err)
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.WriteHeader(HTTPStatus(appErr.Code))
        enc := json.NewEncoder(w)
        enc.SetEscapeHTML(false)
        _ = enc.Encode(errorEnvelope{Error: errorBody{
                Code:    appErr.Code,
                Message: appErr.Message,
                Details: appErr.Details,
        }})
}

// WriteJSON отвечает произвольным payload с указанным статусом.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.WriteHeader(status)
        enc := json.NewEncoder(w)
        enc.SetEscapeHTML(false)
        _ = enc.Encode(payload)
}