// Package auth реализует mTLS-аутентификацию и роли (ADR-006).
package auth

import "context"

// Role — роль устройства, определяется по CN клиентского сертификата.
type Role string

const (
	// RoleWeb — ПК-админка: полный CRUD всех модулей.
	RoleWeb Role = "web"
	// RoleAndroid — Android-клиент: чтение + комментарии,
	// запись только в замкнутые модули.
	RoleAndroid Role = "android"
	// RoleUnknown — сертификат не распознан.
	RoleUnknown Role = "unknown"
)

// RoleFromCN сопоставляет Common Name сертификата с ролью.
// webCN и androidCN — ожидаемые CN из конфига (ADR-006).
func RoleFromCN(cn, webCN, androidCN string) Role {
	switch cn {
	case webCN:
		return RoleWeb
	case androidCN:
		return RoleAndroid
	default:
		return RoleUnknown
	}
}

type ctxKey int

const roleCtxKey ctxKey = iota

// ContextWithRole сохраняет роль в контексте запроса.
func ContextWithRole(ctx context.Context, role Role) context.Context {
	return context.WithValue(ctx, roleCtxKey, role)
}

// RoleFromContext возвращает роль из контекста запроса
// (RoleUnknown, если роль не установлена).
func RoleFromContext(ctx context.Context) Role {
	if role, ok := ctx.Value(roleCtxKey).(Role); ok {
		return role
	}
	return RoleUnknown
}
