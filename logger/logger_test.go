package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"redteamcoin/config"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "info level text format",
			cfg: Config{
				Level:  "info",
				Format: "text",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "debug level json format",
			cfg: Config{
				Level:  "debug",
				Format: "json",
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "warn level with verbose",
			cfg: Config{
				Level:   "warn",
				Format:  "text",
				Verbose: true, // should override to debug
				Output:  &bytes.Buffer{},
			},
		},
		{
			name: "info level with quiet",
			cfg: Config{
				Level:  "info",
				Format: "text",
				Quiet:  true, // should override to error
				Output: &bytes.Buffer{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.cfg)
			if logger == nil {
				t.Error("New() returned nil logger")
			}
		})
	}
}

func TestOutputFormats(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedString string
	}{
		{
			name:           "text format",
			format:         "text",
			expectedString: "level=INFO msg=\"test message\"",
		},
		{
			name:           "json format",
			format:         "json",
			expectedString: `"msg":"test message"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(Config{
				Level:  "info",
				Format: tt.format,
				Output: buf,
			})

			logger.Info("test message")

			output := buf.String()
			if !strings.Contains(output, tt.expectedString) {
				t.Errorf("Output format %q doesn't contain expected string %q\nGot: %s",
					tt.format, tt.expectedString, output)
			}
		})
	}
}

func TestLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		level         string
		logFunc       func(*slog.Logger)
		shouldAppear  bool
		messageMarker string
	}{
		{
			name:  "info level shows info",
			level: "info",
			logFunc: func(l *slog.Logger) {
				l.Info("info message")
			},
			shouldAppear:  true,
			messageMarker: "info message",
		},
		{
			name:  "info level hides debug",
			level: "info",
			logFunc: func(l *slog.Logger) {
				l.Debug("debug message")
			},
			shouldAppear:  false,
			messageMarker: "debug message",
		},
		{
			name:  "warn level hides info",
			level: "warn",
			logFunc: func(l *slog.Logger) {
				l.Info("info message")
			},
			shouldAppear:  false,
			messageMarker: "info message",
		},
		{
			name:  "error level shows errors",
			level: "error",
			logFunc: func(l *slog.Logger) {
				l.Error("error message")
			},
			shouldAppear:  true,
			messageMarker: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(Config{
				Level:  tt.level,
				Format: "text",
				Output: buf,
			})
			tt.logFunc(logger)

			output := buf.String()
			contains := strings.Contains(output, tt.messageMarker)

			if tt.shouldAppear && !contains {
				t.Errorf("Expected message %q to appear but it didn't\nGot: %s",
					tt.messageMarker, output)
			}
			if !tt.shouldAppear && contains {
				t.Errorf("Expected message %q to be filtered but it appeared\nGot: %s",
					tt.messageMarker, output)
			}
		})
	}
}

func TestQuietMode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "info",
		Format: "text",
		Quiet:  true, // should only show errors
		Output: buf,
	})

	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	if strings.Contains(output, "info message") {
		t.Error("Quiet mode showed info message")
	}
	if strings.Contains(output, "warn message") {
		t.Error("Quiet mode showed warn message")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Quiet mode didn't show error message")
	}
}

func TestVerboseMode(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:   "warn", // normally wouldn't show debug
		Format:  "text",
		Verbose: true, // should override to debug
		Output:  buf,
	})

	logger.Debug("debug message")

	output := buf.String()

	if !strings.Contains(output, "debug message") {
		t.Error("Verbose mode didn't show debug message")
	}
}

func TestContextPropagation(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "info",
		Format: "text",
		Output: buf,
	})

	ctx := context.Background()
	ctx = WithLogger(ctx, logger)

	// Verify logger can be retrieved
	retrieved := FromContext(ctx)
	if retrieved != logger {
		t.Error("Retrieved logger doesn't match stored logger")
	}

	// Test context-aware logging
	InfoContext(ctx, "context info message")

	output := buf.String()
	if !strings.Contains(output, "context info message") {
		t.Error("Context-aware logging didn't work")
	}
}

func TestContextFallback(t *testing.T) {
	// Test that FromContext falls back to global logger
	SetDefault()
	ctx := context.Background() // no logger stored

	logger := FromContext(ctx)
	if logger == nil {
		t.Error("FromContext returned nil when falling back to global")
	}
}

func TestThreadSafety(t *testing.T) {
	var wg sync.WaitGroup
	iterations := 100

	// Test concurrent Get/Set
	for i := 0; i < iterations; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			Get()
		}()

		go func() {
			defer wg.Done()
			buf := &bytes.Buffer{}
			logger := New(Config{
				Level:  "info",
				Format: "text",
				Output: buf,
			})
			Set(logger)
		}()
	}

	wg.Wait()
	// If we get here without panic, thread safety test passed
}

func TestGlobalLoggerFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "debug",
		Format: "text",
		Output: buf,
	})
	Set(logger)

	// Test global logging functions
	Info("info message")
	Warn("warn message")
	Error("error message")
	Debug("debug message")

	output := buf.String()

	messages := []string{"info message", "warn message", "error message", "debug message"}
	for _, msg := range messages {
		if !strings.Contains(output, msg) {
			t.Errorf("Global logging function didn't log %q\nGot: %s", msg, output)
		}
	}
}

func TestNewFromServerConfig(t *testing.T) {
	cfg := &config.ServerConfig{
		Logging: config.LoggingConfig{
			Level:   "debug",
			Format:  "json",
			Quiet:   false,
			Verbose: true,
		},
	}

	logger := NewFromServerConfig(cfg)
	if logger == nil {
		t.Error("NewFromServerConfig returned nil")
	}
}

func TestNewFromClientConfig(t *testing.T) {
	cfg := &config.ClientConfig{
		Logging: config.ClientLoggingConfig{
			Level:   "info",
			Format:  "text",
			Quiet:   false,
			Verbose: false,
		},
	}

	logger := NewFromClientConfig(cfg)
	if logger == nil {
		t.Error("NewFromClientConfig returned nil")
	}
}

func TestLevelParsing(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected slog.Level
	}{
		{
			name:     "debug level",
			cfg:      Config{Level: "debug"},
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			cfg:      Config{Level: "info"},
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			cfg:      Config{Level: "warn"},
			expected: slog.LevelWarn,
		},
		{
			name:     "warning level",
			cfg:      Config{Level: "warning"},
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			cfg:      Config{Level: "error"},
			expected: slog.LevelError,
		},
		{
			name:     "verbose overrides",
			cfg:      Config{Level: "error", Verbose: true},
			expected: slog.LevelDebug,
		},
		{
			name:     "quiet overrides",
			cfg:      Config{Level: "debug", Quiet: true},
			expected: slog.LevelError,
		},
		{
			name:     "invalid defaults to info",
			cfg:      Config{Level: "invalid"},
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := parseLevel(tt.cfg)
			if level != tt.expected {
				t.Errorf("parseLevel() = %v, want %v", level, tt.expected)
			}
		})
	}
}

func TestAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "info",
		Format: "text",
		Output: buf,
	})

	logger.Info("test message",
		"key1", "value1",
		"key2", 42,
		"key3", true)

	output := buf.String()

	// Check that attributes appear in output
	expectedAttrs := []string{"key1=value1", "key2=42", "key3=true"}
	for _, attr := range expectedAttrs {
		if !strings.Contains(output, attr) {
			t.Errorf("Output doesn't contain attribute %q\nGot: %s", attr, output)
		}
	}
}

func TestContextAwareFunctions(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(Config{
		Level:  "debug",
		Format: "text",
		Output: buf,
	})

	ctx := WithLogger(context.Background(), logger)

	// Test all context-aware functions
	InfoContext(ctx, "info context")
	WarnContext(ctx, "warn context")
	ErrorContext(ctx, "error context")
	DebugContext(ctx, "debug context")

	output := buf.String()

	messages := []string{"info context", "warn context", "error context", "debug context"}
	for _, msg := range messages {
		if !strings.Contains(output, msg) {
			t.Errorf("Context-aware logging didn't log %q\nGot: %s", msg, output)
		}
	}
}
