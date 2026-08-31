// Package logging — структурный JSON-логгер с ротацией (ADR-010):
// сообщения на русском, ключи полей на английском.
package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Level — уровень логирования.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String возвращает имя уровня (английское, для машинной обработки).
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLevel разбирает строковое имя уровня.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug, nil
	case "info", "":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, nil
	}
}

// Field — пара ключ/значение для структурированного лога.
type Field struct {
	Key   string
	Value any
}

// F — короткий конструктор поля.
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Logger — потокобезопасный логгер с ротацией файлов.
type Logger struct {
	mu        sync.Mutex
	out       io.Writer
	level     Level
	requestID string
}

// Config — конфигурация логгера.
type Config struct {
	Level     string
	File      string
	MaxSizeMB int
	MaxFiles  int
}

// New создаёт логгер с ротацией через lumberjack.
func New(cfg Config) *Logger {
	lvl, _ := ParseLevel(cfg.Level)

	var w io.Writer = os.Stdout
	if cfg.File != "" && cfg.File != "stdout" {
		maxSize := cfg.MaxSizeMB
		if maxSize == 0 {
			maxSize = 10
		}
		maxFiles := cfg.MaxFiles
		if maxFiles == 0 {
			maxFiles = 5
		}

		w = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    maxSize,
			MaxBackups: maxFiles,
			LocalTime:  true,
			Compress:   false,
		}
	}

	return &Logger{out: w, level: lvl}
}

func (l *Logger) log(lvl Level, msg string, fields ...Field) {
	if lvl < l.level {
		return
	}

	entry := make(map[string]any, len(fields)+4)
	entry["ts"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	entry["level"] = lvl.String()
	entry["msg"] = msg
	if l.requestID != "" {
		entry["request_id"] = l.requestID
	}
	for _, f := range fields {
		entry[f.Key] = f.Value
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	enc := json.NewEncoder(l.out)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}

// Debug — отладочное сообщение.
func (l *Logger) Debug(msg string, fields ...Field) { l.log(LevelDebug, msg, fields...) }

// Info — информационное сообщение.
func (l *Logger) Info(msg string, fields ...Field) { l.log(LevelInfo, msg, fields...) }

// Warn — предупреждение.
func (l *Logger) Warn(msg string, fields ...Field) { l.log(LevelWarn, msg, fields...) }

// Error — ошибка.
func (l *Logger) Error(msg string, fields ...Field) { l.log(LevelError, msg, fields...) }

// WithRequestID возвращает копию логгера с добавленным request_id.
func (l *Logger) WithRequestID(ctx context.Context) *Logger {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return &Logger{
			out:       l.out,
			level:     l.level,
			requestID: reqID,
		}
	}
	return l
}

type contextKey string

const RequestIDKey contextKey = "request_id"

// ContextWithRequestID добавляет request_id в контекст.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
