package codingagent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/myersguo/cc-mono/pkg/ai"
)

// ModelConfig represents a model configuration from models.json
type ModelConfig struct {
	ID               string  `json:"id"`
	Provider         string  `json:"provider"`
	Name             string  `json:"name"`
	ContextWindow    int     `json:"context_window"`
	MaxOutput        int     `json:"max_output"`
	InputCostPer1M   float64 `json:"input_cost_per_million"`
	OutputCostPer1M  float64 `json:"output_cost_per_million"`
	SupportsVision   bool    `json:"supports_vision"`
	SupportsTools    bool    `json:"supports_tools"`
	SupportsThinking bool    `json:"supports_thinking,omitempty"`
}

// ModelsFile represents the models.json file structure
type ModelsFile struct {
	Models []ModelConfig `json:"models"`
}

// ModelRegistry manages model configurations
type ModelRegistry struct {
	models map[string]ModelConfig
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]ModelConfig),
	}
}

// LoadFromFile loads models from a JSON file
func (r *ModelRegistry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read models file: %w", err)
	}

	var modelsFile ModelsFile
	if err := json.Unmarshal(data, &modelsFile); err != nil {
		return fmt.Errorf("failed to parse models file: %w", err)
	}

	// Add models to registry
	for _, modelConfig := range modelsFile.Models {
		r.models[modelConfig.ID] = modelConfig
	}

	return nil
}

// Get retrieves a model configuration by ID
func (r *ModelRegistry) Get(id string) (ModelConfig, error) {
	model, ok := r.models[id]
	if !ok {
		return ModelConfig{}, fmt.Errorf("model not found: %s", id)
	}
	return model, nil
}

// List returns all registered models
func (r *ModelRegistry) List() []ModelConfig {
	models := make([]ModelConfig, 0, len(r.models))
	for _, model := range r.models {
		models = append(models, model)
	}
	return models
}

// ListByProvider returns all models for a specific provider
func (r *ModelRegistry) ListByProvider(provider string) []ModelConfig {
	models := make([]ModelConfig, 0)
	for _, model := range r.models {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// ToAIModel converts a ModelConfig to an ai.Model
func (r *ModelRegistry) ToAIModel(id string) (ai.Model, error) {
	config, err := r.Get(id)
	if err != nil {
		return ai.Model{}, err
	}

	thinkingLevel := ai.ThinkingLevelNone
	if config.SupportsThinking {
		thinkingLevel = ai.ThinkingLevelMedium
	}

	return ai.Model{
		ID:               config.ID,
		Provider:         config.Provider,
		Name:             config.Name,
		ContextWindow:    config.ContextWindow,
		MaxOutput:        config.MaxOutput,
		InputCostPer1M:   config.InputCostPer1M,
		OutputCostPer1M:  config.OutputCostPer1M,
		SupportsVision:   config.SupportsVision,
		SupportsTools:    config.SupportsTools,
		SupportsThinking: config.SupportsThinking,
		ThinkingLevel:    thinkingLevel,
	}, nil
}

// Register manually registers a model
func (r *ModelRegistry) Register(config ModelConfig) {
	r.models[config.ID] = config
}

// ProviderConfig represents configuration for a provider
type ProviderConfig struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url,omitempty"`
	DefaultModel string `json:"default_model,omitempty"`
}

// ProvidersConfig represents the providers configuration
type ProvidersConfig struct {
	Providers       map[string]ProviderConfig `json:"providers"`
	DefaultProvider string                    `json:"default_provider,omitempty"`
}

// LoadProvidersConfig loads provider configurations from a JSON file
func LoadProvidersConfig(path string) (*ProvidersConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers config: %w", err)
	}

	var config ProvidersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse providers config: %w", err)
	}

	// Expand environment variables in API keys
	for name, providerConfig := range config.Providers {
		providerConfig.APIKey = expandEnvVars(providerConfig.APIKey)
		config.Providers[name] = providerConfig
	}

	return &config, nil
}

// expandEnvVars expands environment variables in the format ${VAR_NAME}
func expandEnvVars(s string) string {
	if !strings.Contains(s, "${") {
		return s
	}

	// Simple environment variable expansion
	result := s
	for {
		start := strings.Index(result, "${")
		if start < 0 {
			break
		}

		end := strings.Index(result[start:], "}")
		if end < 0 {
			break
		}
		end += start

		varName := result[start+2 : end]
		varValue := os.Getenv(varName)

		result = result[:start] + varValue + result[end+1:]
	}

	return result
}
