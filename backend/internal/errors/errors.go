// Package errors — доменные ошибки приложения (ADR-010):
// сообщения — на русском, коды — на английском (для grep и машинной обработки).
package errors

import (
        stderrors "errors"
        "fmt"
		"net/http"
)

// Code — машинный код ошибки (английский).
type Code string

const (
		CodeBadRequest      Code = "bad_request"
        CodeUnauthorized    Code = "unauthorized"
        CodeForbidden       Code = "forbidden"
        CodeNotFound        Code = "not_found"
        CodeValidation      Code = "validation"
        CodeConflict        Code = "conflict"
        CodeDuplicate       Code = "duplicate"
        CodeRateLimited     Code = "rate_limited"
        CodePayloadTooLarge Code = "payload_too_large"
        CodeInternal        Code = "internal"
)

// HTTPStatus возвращает HTTP-статус для кода ошибки.
func (c Code) HTTPStatus() int {
        switch c {
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
        case CodePayloadTooLarge:
                return http.StatusRequestEntityTooLarge
        case CodeInternal:
                return http.StatusInternalServerError
        default:
                return http.StatusInternalServerError
        }
}

// AppError — ошибка приложения с пользовательским сообщением на русском.
type AppError struct {
        Code    Code   `json:"code"`
        Message string `json:"message"`
        Details any    `json:"details,omitempty"`
        Wrapped error  `json:"-"`
}

// Envelope — обёртка для JSON-ответа с ошибкой.
type Envelope struct {
        Error *AppError `json:"error"`
}

func (e *AppError) Error() string {
        if e.Wrapped != nil {
                return fmt.Sprintf("%s: %v", e.Message, e.Wrapped)
        }
        return e.Message
}

func (e *AppError) Unwrap() error { return e.Wrapped }

// Envelope возвращает обёртку для JSON-ответа.
func (e *AppError) Envelope() *Envelope {
        return &Envelope{Error: e}
}

// HTTPStatus возвращает HTTP-статус для AppError.
func (e *AppError) HTTPStatus() int {
	return e.Code.HTTPStatus()
}

func newErr(code Code, message string) *AppError {
        return &AppError{Code: code, Message: message}
}

func withMsg(code Code, def string, msg []string) *AppError {
        if len(msg) > 0 && msg[0] != "" {
                return newErr(code, msg[0])
        }
        return newErr(code, def)
}

// BadRequest — 400: неверный запрос (отсутствует обязательный параметр).
func BadRequest(msg ...string) *AppError {
        return withMsg(CodeBadRequest, "некорректный запрос", msg)
}

// Unauthorized — 401: отсутствует или не распознан клиентский сертификат.
func Unauthorized(msg ...string) *AppError {
        return withMsg(CodeUnauthorized, "требуется клиентский сертификат", msg)
}

// Forbidden — 403: роль не имеет прав на операцию.
func Forbidden(msg ...string) *AppError {
        return withMsg(CodeForbidden, "недостаточно прав для этой операции", msg)
}

// NotFound — 404: запись не найдена.
func NotFound(msg ...string) *AppError {
        return withMsg(CodeNotFound, "запись не найдена", msg)
}

// Validation — 422: ошибка валидации входных данных (с деталями).
func Validation(message string, details any) *AppError {
        if message == "" {
                message = "ошибка валидации"
        }
        return &AppError{Code: CodeValidation, Message: message, Details: details}
}

// Conflict — 409: конфликт версий при синхронизации (LWW).
func Conflict(msg ...string) *AppError {
        return withMsg(CodeConflict, "конфликт версий: запись была изменена", msg)
}

// Duplicate — 409: нарушение уникальности (имя, артикул и т.п.).
func Duplicate(msg ...string) *AppError {
        return withMsg(CodeDuplicate, "запись с такими данными уже существует", msg)
}

// RateLimited — 429: превышен лимит запросов.
func RateLimited(msg ...string) *AppError {
        return withMsg(CodeRateLimited, "превышен лимит запросов", msg)
}

// PayloadTooLarge — 413: размер запроса превышает лимит.
func PayloadTooLarge(msg ...string) *AppError {
        return withMsg(CodePayloadTooLarge, "размер запроса превышает лимит", msg)
}

// Internal — 500: внутренняя ошибка сервера.
func Internal(msg ...string) *AppError {
        return withMsg(CodeInternal, "внутренняя ошибка сервера", msg)
}

// UnknownModule — 400: неизвестный модуль.
func UnknownModule(msg ...string) *AppError {
        return withMsg(Code("unknown_module"), "неизвестный модуль", msg)
}

// ConflictWithVersion — 409: конфликт версий с указанием версии сервера.
func ConflictWithVersion(serverVersion int64) *AppError {
        return &AppError{
                Code:    CodeConflict,
                Message: fmt.Sprintf("конфликт версий: версия сервера %d", serverVersion),
                Details: map[string]int64{"server_version": serverVersion},
        }
}

// Wrap оборачивает низкоуровневую ошибку в AppError с заданным кодом.
func Wrap(err error, code Code, message string) *AppError {
        return &AppError{Code: code, Message: message, Wrapped: err}
}

// AsAppError извлекает *AppError из цепочки ошибок.
func AsAppError(err error) (*AppError, bool) {
        var appErr *AppError
        if stderrors.As(err, &appErr) {
                return appErr, true
        }
        return nil, false
}

// FromError гарантирует AppError: доменная ошибка возвращается как есть,
// любая другая оборачивается в CodeInternal.
func FromError(err error) *AppError {
        if err == nil {
                return Internal()
        }
        if appErr, ok := AsAppError(err); ok {
                return appErr
        }
        return &AppError{Code: CodeInternal, Message: "внутренняя ошибка сервера", Wrapped: err}
}