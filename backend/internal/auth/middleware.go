package auth

import (
	"net/http"

	apperrors "fitoscout/backend/internal/errors"
)

// RequireClientCertConfig возвращает middleware, который пропускает только
// запросы с распознаваемым клиентским сертификатом.
// webCN и androidCN — ожидаемые Common Name сертификатов из конфига (ADR-006).
func RequireClientCertConfig(webCN, androidCN string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cert := PeerCertificate(r)
			if cert == nil {
				apperrors.WriteError(w, apperrors.Unauthorized())
				return
			}

			role := RoleFromCN(cert.Subject.CommonName, webCN, androidCN)
			if role == RoleUnknown {
				apperrors.WriteError(w, apperrors.Unauthorized("клиентский сертификат не распознан"))
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithRole(r.Context(), role)))
		})
	}
}

// RequireClientCert — обёртка с дефолтными CN (для обратной совместимости).
// В реальном приложении используйте RequireClientCertConfig с CN из конфига.
func RequireClientCert(next http.Handler) http.Handler {
	return RequireClientCertConfig("fitoscout-web-admin", "fitoscout-android-client")(next)
}

// RoleCheck возвращает middleware, пропускающий только перечисленные роли.
// Используется для маршрутов с ограниченным доступом (например, запись
// в справочные модули доступна только роли web).
func RoleCheck(allowed ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			for _, a := range allowed {
				if role == a {
					next.ServeHTTP(w, r)
					return
				}
			}
			apperrors.WriteError(w, apperrors.Forbidden())
		})
	}
}

// VerifyClientHeader — дополнительная проверка заголовка X-Fitoscout-Client
// (ADR-006). Если заголовок присутствует, он должен соответствовать роли
// сертификата.
func VerifyClientHeader(headerName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get(headerName)
			if got != "" {
				var expected string
				switch RoleFromContext(r.Context()) {
				case RoleWeb:
					expected = "web-admin"
				case RoleAndroid:
					expected = "android-client"
				}
				if expected == "" || got != expected {
					apperrors.WriteError(w, apperrors.Forbidden("заголовок клиента не соответствует сертификату"))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
