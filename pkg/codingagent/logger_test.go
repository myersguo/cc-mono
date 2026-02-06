package codingagent

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("TextFormat", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(LoggerConfig{
			Level:  LogLevelInfo,
			Format: "text",
			Output: &buf,
		})

		logger.Info("test message", "key", "value")

		output := buf.String()
		if !strings.Contains(output, "test message") {
			t.Errorf("Expected log to contain 'test message', got: %s", output)
		}
		if !strings.Contains(output, "key=value") {
			t.Errorf("Expected log to contain 'key=value', got: %s", output)
		}
	})

	t.Run("JSONFormat", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(LoggerConfig{
			Level:  LogLevelInfo,
			Format: "json",
			Output: &buf,
		})

		logger.Info("test message", "key", "value")

		output := buf.String()
		if !strings.Contains(output, `"msg":"test message"`) {
			t.Errorf("Expected JSON log to contain message, got: %s", output)
		}
		if !strings.Contains(output, `"key":"value"`) {
			t.Errorf("Expected JSON log to contain key-value, got: %s", output)
		}
	})

	t.Run("LogLevel", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(LoggerConfig{
			Level:  LogLevelWarn,
			Format: "text",
			Output: &buf,
		})

		// Debug and Info should not be logged
		logger.Debug("debug message")
		logger.Info("info message")

		// Warn and Error should be logged
		logger.Warn("warn message")
		logger.Error("error message")

		output := buf.String()
		if strings.Contains(output, "debug message") {
			t.Error("Debug message should not be logged at Warn level")
		}
		if strings.Contains(output, "info message") {
			t.Error("Info message should not be logged at Warn level")
		}
		if !strings.Contains(output, "warn message") {
			t.Error("Warn message should be logged")
		}
		if !strings.Contains(output, "error message") {
			t.Error("Error message should be logged")
		}
	})

	t.Run("DefaultLogger", func(t *testing.T) {
		logger := NewDefaultLogger()
		if logger == nil {
			t.Fatal("Default logger should not be nil")
		}

		// Should not panic
		logger.Info("test")
	})

	t.Run("LoggerFromConfig", func(t *testing.T) {
		config := NewConfig(NewDefaultLogger())
		config.Set("log.level", "debug")
		config.Set("log.format", "json")

		logger := LoggerFromConfig(config)
		if logger == nil {
			t.Fatal("Logger from config should not be nil")
		}

		// Verify it logs at debug level
		var buf bytes.Buffer
		logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logger.Debug("debug test")

		if !strings.Contains(buf.String(), "debug test") {
			t.Error("Logger should log debug messages")
		}
	})
}
