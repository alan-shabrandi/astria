package logger

import (
	"log/slog"
	"os"
)

// Setup initializes a structured JSON logger for the application.
func Setup(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	// JSONHandler outputs structured logs which are optimal for LLMOps and Observability (Day 20)
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	// Set as global default logger
	slog.SetDefault(logger)

	return logger
}
