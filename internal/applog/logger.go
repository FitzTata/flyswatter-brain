package applog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level  string
	Format string
	File   string
}

func New(config Config) (*slog.Logger, io.Closer, error) {
	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, nil, err
	}
	format := strings.ToLower(valueOr(config.Format, "json"))
	if format != "json" && format != "text" {
		return nil, nil, fmt.Errorf("FLYSWATTER_LOG_FORMAT must be json or text")
	}

	writers := []io.Writer{os.Stdout}
	var closer io.Closer
	if config.File != "" {
		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		writers = append(writers, file)
		closer = file
	}

	options := &slog.HandlerOptions{Level: level}
	writer := io.MultiWriter(writers...)
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(writer, options)
	} else {
		handler = slog.NewJSONHandler(writer, options)
	}
	return slog.New(handler).With("component", "server"), closer, nil
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(valueOr(raw, "info")) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("FLYSWATTER_LOG_LEVEL must be debug, info, warn, or error")
	}
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
