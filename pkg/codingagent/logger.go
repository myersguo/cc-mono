package codingagent

import (
	"io"
	"log/slog"
	"os"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level  LogLevel
	Format string // "text" or "json"
	Output io.Writer
}

// NewLogger creates a new structured logger
func NewLogger(config LoggerConfig) *slog.Logger {
	// Default values
	if config.Output == nil {
		config.Output = os.Stderr
	}
	if config.Format == "" {
		config.Format = "text"
	}
	if config.Level == "" {
		config.Level = LogLevelInfo
	}

	// Parse log level
	var level slog.Level
	switch config.Level {
	case LogLevelDebug:
		level = slog.LevelDebug
	case LogLevelInfo:
		level = slog.LevelInfo
	case LogLevelWarn:
		level = slog.LevelWarn
	case LogLevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize attribute formatting if needed
			return a
		},
	}

	// Create handler based on format
	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(config.Output, opts)
	} else {
		handler = slog.NewTextHandler(config.Output, opts)
	}

	return slog.New(handler)
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger() *slog.Logger {
	return NewLogger(LoggerConfig{
		Level:  LogLevelInfo,
		Format: "text",
		Output: os.Stderr,
	})
}

// LoggerFromConfig creates a logger from configuration
func LoggerFromConfig(config *Config) *slog.Logger {
	level := LogLevel(config.GetString("log.level"))
	if level == "" {
		level = LogLevelInfo
	}

	format := config.GetString("log.format")
	if format == "" {
		format = "text"
	}

	return NewLogger(LoggerConfig{
		Level:  level,
		Format: format,
		Output: os.Stderr,
	})
}
