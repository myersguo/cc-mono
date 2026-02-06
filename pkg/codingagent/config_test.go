package codingagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	logger := NewDefaultLogger()

	t.Run("LoadJSONFile", func(t *testing.T) {
		// Create a temporary JSON config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		configData := `{
			"app": {
				"name": "test-app",
				"version": "1.0.0"
			},
			"server": {
				"port": 8080
			}
		}`

		if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		config := NewConfig(logger)
		if err := config.LoadFile(configPath); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if config.GetString("app.name") != "test-app" {
			t.Errorf("Expected app.name='test-app', got '%s'", config.GetString("app.name"))
		}

		if config.GetInt("server.port") != 8080 {
			t.Errorf("Expected server.port=8080, got %d", config.GetInt("server.port"))
		}
	})

	t.Run("LoadNonexistentFile", func(t *testing.T) {
		config := NewConfig(logger)
		err := config.LoadFile("/nonexistent/config.json")
		if err != nil {
			t.Errorf("Expected no error for nonexistent file, got %v", err)
		}
	})

	t.Run("SetAndGet", func(t *testing.T) {
		config := NewConfig(logger)
		config.Set("test.key", "test-value")

		if config.GetString("test.key") != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", config.GetString("test.key"))
		}
	})

	t.Run("Unmarshal", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		configData := `{
			"database": {
				"host": "localhost",
				"port": 5432,
				"name": "testdb"
			}
		}`

		if err := os.WriteFile(configPath, []byte(configData), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		config := NewConfig(logger)
		if err := config.LoadFile(configPath); err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		type DatabaseConfig struct {
			Host string `koanf:"host"`
			Port int    `koanf:"port"`
			Name string `koanf:"name"`
		}

		var dbConfig DatabaseConfig
		if err := config.Unmarshal("database", &dbConfig); err != nil {
			t.Fatalf("Failed to unmarshal config: %v", err)
		}

		if dbConfig.Host != "localhost" {
			t.Errorf("Expected host='localhost', got '%s'", dbConfig.Host)
		}
		if dbConfig.Port != 5432 {
			t.Errorf("Expected port=5432, got %d", dbConfig.Port)
		}
	})
}
