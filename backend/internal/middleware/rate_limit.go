package middleware

import (
	"net/http"
	"sync"
	"time"

	"fitoscout/backend/internal/auth"
	apperrors "fitoscout/backend/internal/errors"
)

type bucket struct {
	count       int
	windowStart time.Time
}

// RateLimiter — простое скользящее окно «запросов в минуту» на клиента
// (ключ — CN сертификата, при отсутствии — RemoteAddr).
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

// NewRateLimiter создаёт ограничитель: limitPerMin запросов в минуту
// (0 или меньше — ограничение выключено).
func NewRateLimiter(limitPerMin int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limitPerMin,
		window:  time.Minute,
	}
}

// Middleware возвращает HTTP-middleware ограничения запросов.
func (rl *RateLimiter) Middleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl.limit <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			key := auth.ExtractCN(r)
			if key == "" {
				key = r.RemoteAddr
			}

			rl.mu.Lock()
			now := time.Now()
			b, ok := rl.buckets[key]
			if !ok || now.Sub(b.windowStart) >= rl.window {
				b = &bucket{windowStart: now}
				rl.buckets[key] = b
			}
			b.count++
			allowed := b.count <= rl.limit
			rl.mu.Unlock()

			if !allowed {
				apperrors.WriteError(w, apperrors.RateLimited())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
