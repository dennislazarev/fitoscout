package middleware

import (
        "net/http"
        "time"

        "fitoscout/backend/internal/logging"
)

// statusWriter перехватывает статус и размер ответа для логирования.
type statusWriter struct {
        http.ResponseWriter
        status int
        bytes  int
}

func (sw *statusWriter) WriteHeader(code int) {
        sw.status = code
        sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
        if sw.status == 0 {
                sw.status = http.StatusOK
        }
        n, err := sw.ResponseWriter.Write(b)
        sw.bytes += n
        return n, err
}

// Logging логирует каждый запрос: метод, путь, статус, длительность
// (сообщение на русском, ключи на английском — ADR-010).
func Logging(logger *logging.Logger) Middleware {
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        start := time.Now()
                        sw := &statusWriter{ResponseWriter: w}

                        next.ServeHTTP(sw, r)

                        logger.Info("запрос выполнен",
                                logging.F("method", r.Method),
                                logging.F("path", r.URL.Path),
                                logging.F("status", sw.status),
                                logging.F("duration_ms", time.Since(start).Milliseconds()),
                                logging.F("bytes", sw.bytes),
                                logging.F("request_id", FromContext(r.Context())),
                        )
                })
        }
}