package extensions

import (
	"context"
	"testing"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/myersguo/cc-mono/pkg/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExtension for testing
type MockExtension struct {
	*shared.BaseExtension
	onToolCallCalled     bool
	onToolResultCalled   bool
	onAgentStartCalled   bool
	onAgentEndCalled     bool
	modifyParams         bool
	modifyResult         bool
	returnError          bool
	registeredToolsCount int
}

func NewMockExtension(name string) *MockExtension {
	return &MockExtension{
		BaseExtension: shared.NewBaseExtension(name, "1.0.0", "Mock extension for testing"),
	}
}

func (m *MockExtension) OnToolCall(ctx context.Context, toolName string, params map[string]any) (map[string]any, error) {
	m.onToolCallCalled = true

	if m.returnError {
		return nil, assert.AnError
	}

	if m.modifyParams {
		modified := make(map[string]any)
		for k, v := range params {
			modified[k] = v
		}
		modified["modified_by"] = m.Name()
		return modified, nil
	}

	return nil, nil
}

func (m *MockExtension) OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error) {
	m.onToolResultCalled = true

	if m.returnError {
		return result, assert.AnError
	}

	if m.modifyResult {
		return agent.AgentToolResult{
			Content: []ai.Content{ai.NewTextContent("Modified by " + m.Name())},
			IsError: false,
		}, nil
	}

	return agent.AgentToolResult{}, nil
}

func (m *MockExtension) OnAgentStart(ctx context.Context) error {
	m.onAgentStartCalled = true
	if m.returnError {
		return assert.AnError
	}
	return nil
}

func (m *MockExtension) OnAgentEnd(ctx context.Context) error {
	m.onAgentEndCalled = true
	if m.returnError {
		return assert.AnError
	}
	return nil
}

func (m *MockExtension) RegisterTools() []agent.AgentTool {
	tools := make([]agent.AgentTool, m.registeredToolsCount)
	for i := 0; i < m.registeredToolsCount; i++ {
		tools[i] = agent.AgentTool{
			Tool: ai.NewTool(
				"mock-tool",
				"Mock tool",
				map[string]any{},
			),
			Label:   "Mock Tool",
			Execute: nil,
		}
	}
	return tools
}

// Test Loader

func TestLoader_LoadExtension(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	config := map[string]any{
		"key": "value",
	}

	err := loader.LoadExtension(ext, config)
	require.NoError(t, err)

	// Verify extension is loaded
	loaded, exists := loader.GetExtension("test-ext")
	assert.True(t, exists)
	assert.Equal(t, ext, loaded)

	// Verify config is stored
	storedConfig, exists := loader.GetConfig("test-ext")
	assert.True(t, exists)
	assert.Equal(t, config, storedConfig)
}

func TestLoader_LoadExtensionTwice(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	err := loader.LoadExtension(ext, nil)
	require.NoError(t, err)

	// Try to load again
	err = loader.LoadExtension(ext, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already loaded")
}

func TestLoader_UnloadExtension(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	err := loader.LoadExtension(ext, nil)
	require.NoError(t, err)

	// Unload extension
	err = loader.UnloadExtension("test-ext")
	require.NoError(t, err)

	// Verify extension is unloaded
	_, exists := loader.GetExtension("test-ext")
	assert.False(t, exists)
}

func TestLoader_UnloadNonExistent(t *testing.T) {
	loader := NewLoader()

	err := loader.UnloadExtension("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not loaded")
}

func TestLoader_ListExtensions(t *testing.T) {
	loader := NewLoader()

	ext1 := NewMockExtension("ext1")
	ext2 := NewMockExtension("ext2")

	loader.LoadExtension(ext1, nil)
	loader.LoadExtension(ext2, nil)

	exts := loader.ListExtensions()
	assert.Len(t, exts, 2)
}

func TestLoader_UnloadAll(t *testing.T) {
	loader := NewLoader()

	ext1 := NewMockExtension("ext1")
	ext2 := NewMockExtension("ext2")

	loader.LoadExtension(ext1, nil)
	loader.LoadExtension(ext2, nil)

	err := loader.UnloadAll()
	require.NoError(t, err)

	exts := loader.ListExtensions()
	assert.Len(t, exts, 0)
}

func TestLoader_ReloadExtension(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	// Load with initial config
	initialConfig := map[string]any{"version": "1"}
	err := loader.LoadExtension(ext, initialConfig)
	require.NoError(t, err)

	// Reload with new config
	newConfig := map[string]any{"version": "2"}
	err = loader.ReloadExtension("test-ext", newConfig)
	require.NoError(t, err)

	// Verify new config
	storedConfig, _ := loader.GetConfig("test-ext")
	assert.Equal(t, newConfig, storedConfig)
}

// Test Runner

func TestRunner_OnToolCall(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")
	ext.modifyParams = true

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	params := map[string]any{"original": "value"}
	modified, err := runner.OnToolCall(context.Background(), "test-tool", params)

	require.NoError(t, err)
	assert.True(t, ext.onToolCallCalled)
	assert.Contains(t, modified, "modified_by")
	assert.Equal(t, "test-ext", modified["modified_by"])
}

func TestRunner_OnToolCall_Error(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")
	ext.returnError = true

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	_, err := runner.OnToolCall(context.Background(), "test-tool", map[string]any{})
	assert.Error(t, err)
}

func TestRunner_OnToolResult(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")
	ext.modifyResult = true

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	result := agent.AgentToolResult{
		Content: []ai.Content{ai.NewTextContent("Original")},
	}

	modified, err := runner.OnToolResult(context.Background(), "test-tool", result)

	require.NoError(t, err)
	assert.True(t, ext.onToolResultCalled)
	require.Len(t, modified.Content, 1)

	textContent, ok := modified.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Modified by")
}

func TestRunner_OnAgentStart(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	err := runner.OnAgentStart(context.Background())
	require.NoError(t, err)
	assert.True(t, ext.onAgentStartCalled)
}

func TestRunner_OnAgentEnd(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	err := runner.OnAgentEnd(context.Background())
	require.NoError(t, err)
	assert.True(t, ext.onAgentEndCalled)
}

func TestRunner_GetRegisteredTools(t *testing.T) {
	loader := NewLoader()

	ext1 := NewMockExtension("ext1")
	ext1.registeredToolsCount = 2

	ext2 := NewMockExtension("ext2")
	ext2.registeredToolsCount = 3

	loader.LoadExtension(ext1, nil)
	loader.LoadExtension(ext2, nil)

	runner := NewRunner(loader)

	tools := runner.GetRegisteredTools()
	assert.Len(t, tools, 5) // 2 + 3
}

func TestRunner_SetEnabled(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	// Disable runner
	runner.SetEnabled(false)
	assert.False(t, runner.IsEnabled())

	// OnToolCall should be skipped
	_, err := runner.OnToolCall(context.Background(), "test-tool", map[string]any{})
	require.NoError(t, err)
	assert.False(t, ext.onToolCallCalled)

	// Re-enable
	runner.SetEnabled(true)
	assert.True(t, runner.IsEnabled())

	// Now it should work
	_, err = runner.OnToolCall(context.Background(), "test-tool", map[string]any{})
	require.NoError(t, err)
	assert.True(t, ext.onToolCallCalled)
}

func TestRunner_SetToolFilter(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	// Set filter to only intercept "read" tool
	runner.SetToolFilter([]string{"read"})

	// Call with "read" - should be intercepted
	_, err := runner.OnToolCall(context.Background(), "read", map[string]any{})
	require.NoError(t, err)
	assert.True(t, ext.onToolCallCalled)

	// Reset for next test
	ext.onToolCallCalled = false

	// Call with "write" - should be skipped
	_, err = runner.OnToolCall(context.Background(), "write", map[string]any{})
	require.NoError(t, err)
	assert.False(t, ext.onToolCallCalled)
}

func TestRunner_WrapTool(t *testing.T) {
	loader := NewLoader()
	ext := NewMockExtension("test-ext")
	ext.modifyParams = true
	ext.modifyResult = true

	loader.LoadExtension(ext, nil)

	runner := NewRunner(loader)

	// Create a mock tool
	originalExecuted := false
	tool := agent.AgentTool{
		Tool: ai.NewTool("test-tool", "Test tool", map[string]any{}),
		Label: "Test Tool",
		Execute: func(ctx context.Context, toolCallID string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			originalExecuted = true
			assert.Contains(t, params, "modified_by") // Params should be modified
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent("Original result")},
			}, nil
		},
	}

	// Wrap the tool
	wrapped := runner.WrapTool(tool)

	// Execute wrapped tool
	result, err := wrapped.Execute(context.Background(), "call-1", map[string]any{"key": "value"}, nil)

	require.NoError(t, err)
	assert.True(t, originalExecuted)
	assert.True(t, ext.onToolCallCalled)
	assert.True(t, ext.onToolResultCalled)

	// Result should be modified by extension
	require.Len(t, result.Content, 1)
	textContent, ok := result.Content[0].(ai.TextContent)
	require.True(t, ok)
	assert.Contains(t, textContent.Text, "Modified by")
}
