package ai

import (
	"fmt"
	"sync"
)

// ProviderRegistry manages the registration and retrieval of LLM providers
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider
func (r *ProviderRegistry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// List returns all registered provider names
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Unregister removes a provider from the registry
func (r *ProviderRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	delete(r.providers, name)
	return nil
}

// Global registry instance
var globalRegistry = NewProviderRegistry()

// RegisterProvider registers a provider in the global registry
func RegisterProvider(provider Provider) error {
	return globalRegistry.Register(provider)
}

// GetProvider retrieves a provider from the global registry
func GetProvider(name string) (Provider, error) {
	return globalRegistry.Get(name)
}

// ListProviders returns all registered provider names from the global registry
func ListProviders() []string {
	return globalRegistry.List()
}

// UnregisterProvider removes a provider from the global registry
func UnregisterProvider(name string) error {
	return globalRegistry.Unregister(name)
}

// ModelRegistry manages model definitions
type ModelRegistry struct {
	mu     sync.RWMutex
	models map[string]Model
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]Model),
	}
}

// Register registers a model
func (r *ModelRegistry) Register(model Model) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[model.ID]; exists {
		return fmt.Errorf("model %s already registered", model.ID)
	}

	r.models[model.ID] = model
	return nil
}

// Get retrieves a model by ID
func (r *ModelRegistry) Get(id string) (Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.models[id]
	if !exists {
		return Model{}, fmt.Errorf("model %s not found", id)
	}

	return model, nil
}

// List returns all registered models
func (r *ModelRegistry) List() []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]Model, 0, len(r.models))
	for _, model := range r.models {
		models = append(models, model)
	}
	return models
}

// ListByProvider returns all models for a specific provider
func (r *ModelRegistry) ListByProvider(provider string) []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]Model, 0)
	for _, model := range r.models {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// Unregister removes a model from the registry
func (r *ModelRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[id]; !exists {
		return fmt.Errorf("model %s not found", id)
	}

	delete(r.models, id)
	return nil
}

// Global model registry instance
var globalModelRegistry = NewModelRegistry()

// RegisterModel registers a model in the global registry
func RegisterModel(model Model) error {
	return globalModelRegistry.Register(model)
}

// GetModel retrieves a model from the global registry
func GetModel(id string) (Model, error) {
	return globalModelRegistry.Get(id)
}

// ListModels returns all registered models from the global registry
func ListModels() []Model {
	return globalModelRegistry.List()
}

// ListModelsByProvider returns all models for a provider from the global registry
func ListModelsByProvider(provider string) []Model {
	return globalModelRegistry.ListByProvider(provider)
}

// UnregisterModel removes a model from the global registry
func UnregisterModel(id string) error {
	return globalModelRegistry.Unregister(id)
}
