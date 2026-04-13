package logger

import (
	"fmt"
	"log/slog"
	"os"
)

func InitLogger(level string, pretty bool, logDir string) (*slog.Logger, error) {
	logLevel := slog.LevelInfo

	switch level {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		fmt.Printf("Unknown log level %s, defaulting to INFO\n", level)
		logLevel = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var TerminalHandler slog.Handler

	if pretty {
		TerminalHandler = NewHandler(WithColor(true), WithLevel(logLevel), WithEncoder(JSON), WithWriter(os.Stdout))
	} else {
		TerminalHandler = slog.NewJSONHandler(os.Stdout, opts)
	}
	var fileHandler slog.Handler

	if logDir != "" {
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			if err := os.MkdirAll(logDir, 0o755); err != nil {
				return nil, fmt.Errorf("failed to create log directory: %w", err)
			}
		}
		logFile := fmt.Sprintf("%s/shipment_service.log", logDir)
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		fileHandler = slog.NewJSONHandler(f, opts)
	} else {
		fileHandler = slog.NewJSONHandler(os.Stdout, opts)
	}
	log := slog.NewMultiHandler(TerminalHandler, fileHandler)

	return slog.New(log), nil
}
