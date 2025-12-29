package logger

import (
	"context"
	"log/slog"
)

// contextKey is the type for logger context keys
type contextKey string

const loggerKey contextKey = "logger"

// WithLogger stores a logger in the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves a logger from the context, or returns the global logger if not found
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return Get()
}

// InfoContext logs an informational message using the logger from context
func InfoContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning message using the logger from context
func WarnContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error message using the logger from context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).ErrorContext(ctx, msg, args...)
}

// DebugContext logs a debug message using the logger from context
func DebugContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).DebugContext(ctx, msg, args...)
}
