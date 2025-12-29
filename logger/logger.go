// Package logger provides production-grade structured logging for RedTeamCoin.
//
// It uses Go's standard log/slog package with support for multiple output formats
// (text, color, JSON), configurable log levels, and context-aware logging.
//
// The logger is thread-safe and supports hot-reloading of configuration.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"redteamcoin/config"
)

// Global logger instance with atomic access for thread safety
var globalLogger atomic.Pointer[slog.Logger]

// Config represents the logger configuration
type Config struct {
	Level   string // debug, info, warn, error
	Format  string // text, color, json
	Quiet   bool   // suppress all but errors
	Verbose bool   // enable debug logs
	Output  io.Writer
}

// Get returns the global logger instance, initializing it with defaults if necessary
func Get() *slog.Logger {
	logger := globalLogger.Load()
	if logger == nil {
		SetDefault()
		logger = globalLogger.Load()
	}
	return logger
}

// Set atomically updates the global logger
func Set(logger *slog.Logger) {
	globalLogger.Store(logger)
}

// SetDefault initializes the global logger with default settings
func SetDefault() {
	logger := New(Config{
		Level:   "info",
		Format:  "text",
		Quiet:   false,
		Verbose: false,
		Output:  os.Stderr,
	})
	Set(logger)
}

// New creates a new logger from the provided configuration
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg)
	handler := createHandler(cfg.Format, level, cfg.Output)
	return slog.New(handler)
}

// NewFromServerConfig creates a logger from RedTeamCoin server configuration
func NewFromServerConfig(cfg *config.ServerConfig) *slog.Logger {
	return New(Config{
		Level:   cfg.Logging.Level,
		Format:  cfg.Logging.Format,
		Quiet:   cfg.Logging.Quiet,
		Verbose: cfg.Logging.Verbose,
		Output:  os.Stderr,
	})
}

// NewFromClientConfig creates a logger from RedTeamCoin client configuration
func NewFromClientConfig(cfg *config.ClientConfig) *slog.Logger {
	return New(Config{
		Level:   cfg.Logging.Level,
		Format:  cfg.Logging.Format,
		Quiet:   cfg.Logging.Quiet,
		Verbose: cfg.Logging.Verbose,
		Output:  os.Stderr,
	})
}

// parseLevel converts string level and flags to slog.Level
func parseLevel(cfg Config) slog.Level {
	// Verbose flag overrides to debug
	if cfg.Verbose {
		return slog.LevelDebug
	}

	// Quiet flag overrides to error only
	if cfg.Quiet {
		return slog.LevelError
	}

	// Parse level string
	level := strings.ToLower(cfg.Level)
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Info logs an informational message using the global logger
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs an error message using the global logger
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// Debug logs a debug message using the global logger
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}
