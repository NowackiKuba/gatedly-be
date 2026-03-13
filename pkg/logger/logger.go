package logger

import (
	"log/slog"
	"os"
)

// New returns a slog.Logger. If env is "production", uses JSONHandler and LevelInfo;
// otherwise uses TextHandler and LevelDebug.
func New(env string) *slog.Logger {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	return slog.New(handler)
}
