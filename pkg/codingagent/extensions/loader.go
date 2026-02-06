package extensions

import (
	"fmt"
	"sync"

	"github.com/myersguo/cc-mono/pkg/shared"
)

// Loader manages loading and initialization of extensions
type Loader struct {
	mu         sync.RWMutex
	extensions map[string]shared.Extension
	configs    map[string]map[string]any
}

// NewLoader creates a new extension loader
func NewLoader() *Loader {
	return &Loader{
		extensions: make(map[string]shared.Extension),
		configs:    make(map[string]map[string]any),
	}
}

// LoadExtension loads and initializes an extension
func (l *Loader) LoadExtension(ext shared.Extension, config map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	name := ext.Name()

	// Check if already loaded
	if _, exists := l.extensions[name]; exists {
		return fmt.Errorf("extension %s is already loaded", name)
	}

	// Initialize extension
	if err := ext.Init(config); err != nil {
		return fmt.Errorf("failed to initialize extension %s: %w", name, err)
	}

	// Store extension and config
	l.extensions[name] = ext
	l.configs[name] = config

	return nil
}

// UnloadExtension unloads an extension
func (l *Loader) UnloadExtension(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	ext, exists := l.extensions[name]
	if !exists {
		return fmt.Errorf("extension %s is not loaded", name)
	}

	// Shutdown extension
	if err := ext.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown extension %s: %w", name, err)
	}

	// Remove from registry
	delete(l.extensions, name)
	delete(l.configs, name)

	return nil
}

// GetExtension retrieves a loaded extension by name
func (l *Loader) GetExtension(name string) (shared.Extension, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	ext, exists := l.extensions[name]
	return ext, exists
}

// ListExtensions returns all loaded extensions
func (l *Loader) ListExtensions() []shared.Extension {
	l.mu.RLock()
	defer l.mu.RUnlock()

	exts := make([]shared.Extension, 0, len(l.extensions))
	for _, ext := range l.extensions {
		exts = append(exts, ext)
	}
	return exts
}

// GetConfig returns the configuration for an extension
func (l *Loader) GetConfig(name string) (map[string]any, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	config, exists := l.configs[name]
	return config, exists
}

// LoadFromRegistry loads extensions from the global registry
func (l *Loader) LoadFromRegistry(names []string, configs map[string]map[string]any) error {
	for _, name := range names {
		ext, ok := shared.GlobalRegistry.Get(name)
		if !ok {
			return fmt.Errorf("extension %s not found in registry", name)
		}

		config := configs[name]
		if config == nil {
			config = make(map[string]any)
		}

		if err := l.LoadExtension(ext, config); err != nil {
			return err
		}
	}

	return nil
}

// UnloadAll unloads all extensions
func (l *Loader) UnloadAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errors []error

	for name, ext := range l.extensions {
		if err := ext.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown %s: %w", name, err))
		}
	}

	// Clear all extensions
	l.extensions = make(map[string]shared.Extension)
	l.configs = make(map[string]map[string]any)

	if len(errors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errors)
	}

	return nil
}

// ReloadExtension reloads an extension with new configuration
func (l *Loader) ReloadExtension(name string, config map[string]any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	ext, exists := l.extensions[name]
	if !exists {
		return fmt.Errorf("extension %s is not loaded", name)
	}

	// Shutdown current instance
	if err := ext.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown extension %s: %w", name, err)
	}

	// Re-initialize with new config
	if err := ext.Init(config); err != nil {
		return fmt.Errorf("failed to re-initialize extension %s: %w", name, err)
	}

	// Update config
	l.configs[name] = config

	return nil
}
