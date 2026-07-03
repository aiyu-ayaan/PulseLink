// Package logging provides the application's structured logger.
//
// It wraps the standard library log/slog so the rest of the backend depends on
// slog directly. There is deliberately no custom logger interface: slog already
// is one.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger writing to stderr at the given level ("debug",
// "info", "warn", "error"). Unknown levels fall back to info.
func New(level string) *slog.Logger {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
