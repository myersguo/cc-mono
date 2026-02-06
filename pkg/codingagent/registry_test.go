package codingagent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRegistry_LoadFromFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test models.json
	modelsJSON := `{
		"models": [
			{
				"id": "gpt-4",
				"provider": "openai",
				"name": "GPT-4",
				"context_window": 8192,
				"max_output": 4096,
				"input_cost_per_million": 30.0,
				"output_cost_per_million": 60.0,
				"supports_vision": false,
				"supports_tools": true
			},
			{
				"id": "claude-3-opus",
				"provider": "anthropic",
				"name": "Claude 3 Opus",
				"context_window": 200000,
				"max_output": 4096,
				"input_cost_per_million": 15.0,
				"output_cost_per_million": 75.0,
				"supports_vision": true,
				"supports_tools": true,
				"supports_thinking": true
			}
		]
	}`

	modelsPath := filepath.Join(tempDir, "models.json")
	err := os.WriteFile(modelsPath, []byte(modelsJSON), 0644)
	require.NoError(t, err)

	// Load models
	registry := NewModelRegistry()
	err = registry.LoadFromFile(modelsPath)
	require.NoError(t, err)

	// Test Get
	model, err := registry.Get("gpt-4")
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", model.ID)
	assert.Equal(t, "openai", model.Provider)
	assert.Equal(t, "GPT-4", model.Name)
	assert.Equal(t, 8192, model.ContextWindow)
	assert.Equal(t, 4096, model.MaxOutput)
	assert.Equal(t, 30.0, model.InputCostPer1M)
	assert.Equal(t, 60.0, model.OutputCostPer1M)
	assert.False(t, model.SupportsVision)
	assert.True(t, model.SupportsTools)
	assert.False(t, model.SupportsThinking)

	// Test Get with thinking support
	model2, err := registry.Get("claude-3-opus")
	require.NoError(t, err)
	assert.Equal(t, "claude-3-opus", model2.ID)
	assert.True(t, model2.SupportsThinking)

	// Test List
	models := registry.List()
	assert.Len(t, models, 2)

	// Test ListByProvider
	openaiModels := registry.ListByProvider("openai")
	assert.Len(t, openaiModels, 1)
	assert.Equal(t, "gpt-4", openaiModels[0].ID)

	anthropicModels := registry.ListByProvider("anthropic")
	assert.Len(t, anthropicModels, 1)
	assert.Equal(t, "claude-3-opus", anthropicModels[0].ID)
}

func TestModelRegistry_GetNotFound(t *testing.T) {
	registry := NewModelRegistry()

	_, err := registry.Get("nonexistent-model")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model not found")
}

func TestModelRegistry_Register(t *testing.T) {
	registry := NewModelRegistry()

	// Manually register a model
	config := ModelConfig{
		ID:               "custom-model",
		Provider:         "custom",
		Name:             "Custom Model",
		ContextWindow:    4096,
		MaxOutput:        2048,
		InputCostPer1M:   10.0,
		OutputCostPer1M:  20.0,
		SupportsVision:   false,
		SupportsTools:    true,
		SupportsThinking: false,
	}

	registry.Register(config)

	// Verify registration
	model, err := registry.Get("custom-model")
	require.NoError(t, err)
	assert.Equal(t, "custom-model", model.ID)
	assert.Equal(t, "custom", model.Provider)
}

func TestModelRegistry_ToAIModel(t *testing.T) {
	registry := NewModelRegistry()

	config := ModelConfig{
		ID:               "test-model",
		Provider:         "test",
		Name:             "Test Model",
		ContextWindow:    8000,
		MaxOutput:        4000,
		InputCostPer1M:   5.0,
		OutputCostPer1M:  10.0,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsThinking: true,
	}

	registry.Register(config)

	// Convert to AI model
	aiModel, err := registry.ToAIModel("test-model")
	require.NoError(t, err)

	assert.Equal(t, "test-model", aiModel.ID)
	assert.Equal(t, "test", aiModel.Provider)
	assert.Equal(t, "Test Model", aiModel.Name)
	assert.Equal(t, 8000, aiModel.ContextWindow)
	assert.Equal(t, 4000, aiModel.MaxOutput)
	assert.Equal(t, 5.0, aiModel.InputCostPer1M)
	assert.Equal(t, 10.0, aiModel.OutputCostPer1M)
	assert.True(t, aiModel.SupportsVision)
	assert.True(t, aiModel.SupportsTools)
	assert.True(t, aiModel.SupportsThinking)
	// Should set thinking level to Medium for models with thinking support
	assert.Equal(t, "medium", string(aiModel.ThinkingLevel))
}

func TestLoadProvidersConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Set environment variable for testing
	os.Setenv("TEST_API_KEY", "test-key-value")
	defer os.Unsetenv("TEST_API_KEY")

	// Create a test providers.json
	providersJSON := `{
		"providers": {
			"openai": {
				"api_key": "${TEST_API_KEY}",
				"base_url": "https://api.openai.com/v1",
				"default_model": "gpt-4"
			},
			"deepseek": {
				"api_key": "hardcoded-key",
				"base_url": "https://api.deepseek.com/v1",
				"default_model": "deepseek-chat"
			}
		},
		"default_provider": "openai"
	}`

	providersPath := filepath.Join(tempDir, "providers.json")
	err := os.WriteFile(providersPath, []byte(providersJSON), 0644)
	require.NoError(t, err)

	// Load providers
	config, err := LoadProvidersConfig(providersPath)
	require.NoError(t, err)

	// Verify default provider
	assert.Equal(t, "openai", config.DefaultProvider)

	// Verify providers
	assert.Len(t, config.Providers, 2)

	// Verify OpenAI provider with environment variable expansion
	openaiConfig, ok := config.Providers["openai"]
	require.True(t, ok)
	assert.Equal(t, "test-key-value", openaiConfig.APIKey) // Environment variable expanded
	assert.Equal(t, "https://api.openai.com/v1", openaiConfig.BaseURL)
	assert.Equal(t, "gpt-4", openaiConfig.DefaultModel)

	// Verify DeepSeek provider with hardcoded key
	deepseekConfig, ok := config.Providers["deepseek"]
	require.True(t, ok)
	assert.Equal(t, "hardcoded-key", deepseekConfig.APIKey)
	assert.Equal(t, "https://api.deepseek.com/v1", deepseekConfig.BaseURL)
	assert.Equal(t, "deepseek-chat", deepseekConfig.DefaultModel)
}

func TestLoadProvidersConfig_FileNotFound(t *testing.T) {
	_, err := LoadProvidersConfig("/nonexistent/providers.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read providers config")
}

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single environment variable",
			input:    "${TEST_VAR}",
			expected: "test-value",
		},
		{
			name:     "Environment variable in string",
			input:    "prefix-${TEST_VAR}-suffix",
			expected: "prefix-test-value-suffix",
		},
		{
			name:     "Multiple environment variables",
			input:    "${TEST_VAR}-${TEST_VAR}",
			expected: "test-value-test-value",
		},
		{
			name:     "No environment variable",
			input:    "plain-string",
			expected: "plain-string",
		},
		{
			name:     "Nonexistent environment variable",
			input:    "${NONEXISTENT_VAR}",
			expected: "", // Empty string for nonexistent variables
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModelRegistry_LoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid JSON file
	invalidJSON := `{invalid json}`
	modelsPath := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(modelsPath, []byte(invalidJSON), 0644)
	require.NoError(t, err)

	registry := NewModelRegistry()
	err = registry.LoadFromFile(modelsPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse models file")
}

func TestLoadProvidersConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Create invalid JSON file
	invalidJSON := `{invalid json}`
	providersPath := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(providersPath, []byte(invalidJSON), 0644)
	require.NoError(t, err)

	_, err = LoadProvidersConfig(providersPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse providers config")
}
