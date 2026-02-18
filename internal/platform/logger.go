package platform

import (
	"log/slog"
	"os"
)

// NewLogger creates a structured logger based on the environment.
func NewLogger(env string) *slog.Logger {
	var handler slog.Handler

	switch env {
	case "production", "staging":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	return slog.New(handler)
}
