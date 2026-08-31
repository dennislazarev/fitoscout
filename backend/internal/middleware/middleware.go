// Package middleware — сквозные HTTP-middleware сервера.
package middleware

import "net/http"

// Middleware — обёртка HTTP-обработчика.
type Middleware func(http.Handler) http.Handler

// Chain оборачивает handler в цепочку middleware.
// Первый элемент списка — самый внешний.
func Chain(handler http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}
