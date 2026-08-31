package middleware

import (
	"fmt"
	"net/http"

	apperrors "fitoscout/backend/internal/errors"
	"fitoscout/backend/internal/logging"
)

// Recovery перехватывает паники в обработчиках, логирует их
// и возвращает клиенту корректную JSON-ошибку 500.
func Recovery(logger *logging.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("паника в обработчике запроса",
						logging.F("panic", fmt.Sprint(rec)),
						logging.F("method", r.Method),
						logging.F("path", r.URL.Path),
						logging.F("request_id", FromContext(r.Context())),
					)
					apperrors.WriteError(w, apperrors.Internal())
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
