package extensions

import (
	"context"
	"fmt"
	"sync"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// Runner executes extension hooks during agent execution
type Runner struct {
	loader     *Loader
	mu         sync.RWMutex
	enabled    bool
	toolFilter map[string]bool // Tools to intercept (empty = all)
}

// NewRunner creates a new extension runner
func NewRunner(loader *Loader) *Runner {
	return &Runner{
		loader:     loader,
		enabled:    true,
		toolFilter: make(map[string]bool),
	}
}

// SetEnabled enables or disables extension execution
func (r *Runner) SetEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = enabled
}

// IsEnabled returns whether extensions are enabled
func (r *Runner) IsEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled
}

// SetToolFilter sets which tools to intercept (empty = all tools)
func (r *Runner) SetToolFilter(tools []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.toolFilter = make(map[string]bool)
	for _, tool := range tools {
		r.toolFilter[tool] = true
	}
}

// shouldIntercept checks if a tool should be intercepted
func (r *Runner) shouldIntercept(toolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If filter is empty, intercept all tools
	if len(r.toolFilter) == 0 {
		return true
	}

	return r.toolFilter[toolName]
}

// OnToolCall executes OnToolCall hooks for all extensions
func (r *Runner) OnToolCall(ctx context.Context, toolName string, params map[string]any) (map[string]any, error) {
	if !r.IsEnabled() || !r.shouldIntercept(toolName) {
		return params, nil
	}

	extensions := r.loader.ListExtensions()
	currentParams := params

	for _, ext := range extensions {
		modifiedParams, err := ext.OnToolCall(ctx, toolName, currentParams)
		if err != nil {
			return nil, fmt.Errorf("extension %s failed on tool call: %w", ext.Name(), err)
		}

		// If extension returns modified params, use them
		if modifiedParams != nil {
			currentParams = modifiedParams
		}
	}

	return currentParams, nil
}

// OnToolResult executes OnToolResult hooks for all extensions
func (r *Runner) OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error) {
	if !r.IsEnabled() || !r.shouldIntercept(toolName) {
		return result, nil
	}

	extensions := r.loader.ListExtensions()
	currentResult := result

	for _, ext := range extensions {
		modifiedResult, err := ext.OnToolResult(ctx, toolName, currentResult)
		if err != nil {
			return result, fmt.Errorf("extension %s failed on tool result: %w", ext.Name(), err)
		}

		// If extension returns a non-zero result, use it
		if len(modifiedResult.Content) > 0 {
			currentResult = modifiedResult
		}
	}

	return currentResult, nil
}

// OnAgentStart executes OnAgentStart hooks for all extensions
func (r *Runner) OnAgentStart(ctx context.Context) error {
	if !r.IsEnabled() {
		return nil
	}

	extensions := r.loader.ListExtensions()

	for _, ext := range extensions {
		if err := ext.OnAgentStart(ctx); err != nil {
			return fmt.Errorf("extension %s failed on agent start: %w", ext.Name(), err)
		}
	}

	return nil
}

// OnAgentEnd executes OnAgentEnd hooks for all extensions
func (r *Runner) OnAgentEnd(ctx context.Context) error {
	if !r.IsEnabled() {
		return nil
	}

	extensions := r.loader.ListExtensions()
	var errors []error

	// Call all extensions even if some fail
	for _, ext := range extensions {
		if err := ext.OnAgentEnd(ctx); err != nil {
			errors = append(errors, fmt.Errorf("extension %s: %w", ext.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during agent end: %v", errors)
	}

	return nil
}

// GetRegisteredTools collects all tools from loaded extensions
func (r *Runner) GetRegisteredTools() []agent.AgentTool {
	extensions := r.loader.ListExtensions()
	var tools []agent.AgentTool

	for _, ext := range extensions {
		extTools := ext.RegisterTools()
		tools = append(tools, extTools...)
	}

	return tools
}

// WrapTool wraps a tool to intercept its execution
func (r *Runner) WrapTool(tool agent.AgentTool) agent.AgentTool {
	originalExecute := tool.Execute

	wrappedExecute := func(
		ctx context.Context,
		toolCallID string,
		params map[string]any,
		onUpdate agent.AgentToolUpdateCallback,
	) (agent.AgentToolResult, error) {
		// Pre-execution hook
		modifiedParams, err := r.OnToolCall(ctx, tool.Tool.Name, params)
		if err != nil {
			return agent.AgentToolResult{
				Content: []ai.Content{ai.NewTextContent(fmt.Sprintf("Extension error: %v", err))},
				IsError: true,
			}, err
		}

		// Execute original tool
		result, err := originalExecute(ctx, toolCallID, modifiedParams, onUpdate)
		if err != nil {
			return result, err
		}

		// Post-execution hook
		modifiedResult, hookErr := r.OnToolResult(ctx, tool.Tool.Name, result)
		if hookErr != nil {
			// Don't fail the tool execution, just log the error
			return result, nil
		}

		if len(modifiedResult.Content) > 0 {
			return modifiedResult, nil
		}

		return result, nil
	}

	return agent.AgentTool{
		Tool:    tool.Tool,
		Label:   tool.Label,
		Execute: wrappedExecute,
	}
}

// WrapAllTools wraps all tools with extension hooks
func (r *Runner) WrapAllTools(tools []agent.AgentTool) []agent.AgentTool {
	wrapped := make([]agent.AgentTool, len(tools))
	for i, tool := range tools {
		wrapped[i] = r.WrapTool(tool)
	}
	return wrapped
}
