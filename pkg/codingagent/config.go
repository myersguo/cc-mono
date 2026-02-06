package codingagent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

// Config represents the application configuration
type Config struct {
	k       *koanf.Koanf
	files   []string
	watcher *fsnotify.Watcher
	logger  *slog.Logger
}

// NewConfig creates a new configuration instance
func NewConfig(logger *slog.Logger) *Config {
	return &Config{
		k:      koanf.New("."),
		files:  make([]string, 0),
		logger: logger,
	}
}

// LoadFile loads a configuration file
func (c *Config) LoadFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.logger.Warn("Config file not found", "path", path)
		return nil
	}

	// Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Determine file type by extension
	ext := strings.ToLower(filepath.Ext(path))
	var parser koanf.Parser

	switch ext {
	case ".yaml", ".yml":
		parser = yaml.Parser()
	case ".json":
		parser = json.Parser()
	default:
		return fmt.Errorf("unsupported config file type: %s", ext)
	}

	// Load the file using rawbytes provider
	if err := c.k.Load(rawbytes.Provider(data), parser); err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	c.files = append(c.files, path)
	c.logger.Info("Loaded config file", "path", path)

	return nil
}

// LoadEnv loads environment variables with a prefix
func (c *Config) LoadEnv(prefix string) error {
	// Load environment variables
	// This is a simplified version - in production, use koanf's env provider
	return nil
}

// Get retrieves a configuration value
func (c *Config) Get(key string) interface{} {
	return c.k.Get(key)
}

// GetString retrieves a string configuration value
func (c *Config) GetString(key string) string {
	return c.k.String(key)
}

// GetInt retrieves an integer configuration value
func (c *Config) GetInt(key string) int {
	return c.k.Int(key)
}

// GetBool retrieves a boolean configuration value
func (c *Config) GetBool(key string) bool {
	return c.k.Bool(key)
}

// GetStringSlice retrieves a string slice configuration value
func (c *Config) GetStringSlice(key string) []string {
	return c.k.Strings(key)
}

// Set sets a configuration value
func (c *Config) Set(key string, value interface{}) error {
	return c.k.Set(key, value)
}

// Unmarshal unmarshals the configuration into a struct
func (c *Config) Unmarshal(key string, target interface{}) error {
	return c.k.Unmarshal(key, target)
}

// UnmarshalAll unmarshals the entire configuration into a struct
func (c *Config) UnmarshalAll(target interface{}) error {
	return c.k.Unmarshal("", target)
}

// Watch starts watching configuration files for changes
func (c *Config) Watch(onChange func()) error {
	if len(c.files) == 0 {
		return fmt.Errorf("no files to watch")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	c.watcher = watcher

	// Add all config files to watcher
	for _, path := range c.files {
		if err := watcher.Add(path); err != nil {
			c.logger.Warn("Failed to watch config file", "path", path, "error", err)
		}
	}

	// Start watching in a goroutine
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					c.logger.Info("Config file changed", "path", event.Name)
					// Reload the file
					if err := c.LoadFile(event.Name); err != nil {
						c.logger.Error("Failed to reload config file", "path", event.Name, "error", err)
					} else {
						onChange()
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				c.logger.Error("Config watcher error", "error", err)
			}
		}
	}()

	return nil
}

// Close closes the configuration watcher
func (c *Config) Close() error {
	if c.watcher != nil {
		return c.watcher.Close()
	}
	return nil
}

// mergeDeep deeply merges src into dest
func mergeDeep(dest, src map[string]interface{}) error {
	for key, srcVal := range src {
		if destVal, exists := dest[key]; exists {
			// If both are maps, merge recursively
			if destMap, destOk := destVal.(map[string]interface{}); destOk {
				if srcMap, srcOk := srcVal.(map[string]interface{}); srcOk {
					if err := mergeDeep(destMap, srcMap); err != nil {
						return err
					}
					continue
				}
			}
		}
		// Otherwise, overwrite
		dest[key] = srcVal
	}
	return nil
}
