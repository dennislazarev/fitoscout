package middleware

import (
        "context"
        "crypto/rand"
        "encoding/hex"
        "net/http"
)

type ctxKey string

// RequestIDKey — ключ ID запроса в контексте.
const RequestIDKey ctxKey = "request_id"

// RequestID проставляет каждому запросу уникальный ID
// (берёт из заголовка X-Request-ID или генерирует новый)
// и возвращает его в ответе.
func RequestID(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                id := r.Header.Get("X-Request-ID")
                if id == "" {
                        id = newRequestID()
                }
                w.Header().Set("X-Request-ID", id)
                next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), RequestIDKey, id)))
        })
}

// FromContext возвращает ID запроса из контекста.
func FromContext(ctx context.Context) string {
        if id, ok := ctx.Value(RequestIDKey).(string); ok {
                return id
        }
        return ""
}

func newRequestID() string {
        b := make([]byte, 8)
        if _, err := rand.Read(b); err != nil {
                return "0000000000000000"
        }
        return hex.EncodeToString(b)
}