package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const FilePath = "storage/logs/app.log"

// New configures console logging and an error-only JSON log file.
// The caller owns the returned file and must close it after the last log call.
func New(environment string, console io.Writer, path string) (*slog.Logger, *os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	var consoleHandler slog.Handler = slog.NewTextHandler(console, options)
	if environment == "production" {
		consoleHandler = slog.NewJSONHandler(console, options)
	}
	fileHandler := slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelError})

	return slog.New(slog.NewMultiHandler(consoleHandler, fileHandler)), file, nil
}
