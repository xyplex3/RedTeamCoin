package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// createHandler creates the appropriate slog.Handler based on format
func createHandler(format string, level slog.Level, output io.Writer) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch format {
	case "json":
		return slog.NewJSONHandler(output, opts)
	case "color":
		if isTerminal(output) {
			return NewColorHandler(output, opts)
		}
		// Fall back to text if not a terminal
		return slog.NewTextHandler(output, opts)
	case "text":
		return slog.NewTextHandler(output, opts)
	default:
		// Default to color if terminal, text otherwise
		if isTerminal(output) {
			return NewColorHandler(output, opts)
		}
		return slog.NewTextHandler(output, opts)
	}
}

// isTerminal checks if the writer is a terminal
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// ColorHandler is a custom slog.Handler that adds ANSI color codes to log output
type ColorHandler struct {
	handler slog.Handler
	output  io.Writer
	opts    *slog.HandlerOptions
}

// NewColorHandler creates a new ColorHandler
func NewColorHandler(output io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ColorHandler{
		handler: slog.NewTextHandler(output, opts),
		output:  output,
		opts:    opts,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle handles the Record with color-coded output
func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	// Format the level with color
	level := colorizeLevel(r.Level)

	// Format the message
	buf := fmt.Sprintf("time=%s level=%s msg=%q",
		r.Time.Format("15:04:05.000"),
		level,
		r.Message)

	// Add attributes
	r.Attrs(func(a slog.Attr) bool {
		buf += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	// Write to output
	_, err := fmt.Fprintln(h.output, buf)
	return err
}

// WithAttrs returns a new Handler with additional attributes
func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{
		handler: h.handler.WithAttrs(attrs),
		output:  h.output,
		opts:    h.opts,
	}
}

// WithGroup returns a new Handler with the given group
func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{
		handler: h.handler.WithGroup(name),
		output:  h.output,
		opts:    h.opts,
	}
}

// colorizeLevel returns the level string with ANSI color codes
func colorizeLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return color.CyanString("DEBUG")
	case slog.LevelInfo:
		return color.GreenString("INFO")
	case slog.LevelWarn:
		return color.YellowString("WARN")
	case slog.LevelError:
		return color.RedString("ERROR")
	default:
		return level.String()
	}
}
