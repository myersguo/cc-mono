package shared

import (
	"context"

	"github.com/myersguo/cc-mono/pkg/agent"
)

// Extension represents a CC-Mono extension
type Extension interface {
	// Name returns the extension name
	Name() string

	// Version returns the extension version
	Version() string

	// Description returns a brief description of the extension
	Description() string

	// Init initializes the extension with configuration
	Init(config map[string]any) error

	// RegisterTools returns tools to register with the agent
	// This is called once during extension initialization
	RegisterTools() []agent.AgentTool

	// OnToolCall is called before a tool is executed
	// Return modified params or nil to skip modification
	// Return error to abort tool execution
	OnToolCall(ctx context.Context, toolName string, params map[string]any) (map[string]any, error)

	// OnToolResult is called after a tool execution
	// Return modified result or nil to skip modification
	// This allows extensions to post-process tool results
	OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error)

	// OnAgentStart is called when the agent starts
	OnAgentStart(ctx context.Context) error

	// OnAgentEnd is called when the agent completes
	OnAgentEnd(ctx context.Context) error

	// Shutdown is called when the extension is being unloaded
	Shutdown() error
}

// BaseExtension provides default implementations for Extension interface
// Embed this in your extension to only override methods you need
type BaseExtension struct {
	name        string
	version     string
	description string
}

// NewBaseExtension creates a new base extension
func NewBaseExtension(name, version, description string) *BaseExtension {
	return &BaseExtension{
		name:        name,
		version:     version,
		description: description,
	}
}

func (e *BaseExtension) Name() string        { return e.name }
func (e *BaseExtension) Version() string     { return e.version }
func (e *BaseExtension) Description() string { return e.description }

func (e *BaseExtension) Init(config map[string]any) error { return nil }
func (e *BaseExtension) RegisterTools() []agent.AgentTool  { return nil }

func (e *BaseExtension) OnToolCall(ctx context.Context, toolName string, params map[string]any) (map[string]any, error) {
	return nil, nil // No modification
}

func (e *BaseExtension) OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error) {
	return agent.AgentToolResult{}, nil // No modification
}

func (e *BaseExtension) OnAgentStart(ctx context.Context) error { return nil }
func (e *BaseExtension) OnAgentEnd(ctx context.Context) error   { return nil }
func (e *BaseExtension) Shutdown() error                        { return nil }

// ExtensionRegistry maintains a registry of available extensions
type ExtensionRegistry struct {
	extensions map[string]Extension
}

// NewExtensionRegistry creates a new extension registry
func NewExtensionRegistry() *ExtensionRegistry {
	return &ExtensionRegistry{
		extensions: make(map[string]Extension),
	}
}

// Register registers an extension
func (r *ExtensionRegistry) Register(ext Extension) {
	r.extensions[ext.Name()] = ext
}

// Get retrieves an extension by name
func (r *ExtensionRegistry) Get(name string) (Extension, bool) {
	ext, ok := r.extensions[name]
	return ext, ok
}

// List returns all registered extensions
func (r *ExtensionRegistry) List() []Extension {
	exts := make([]Extension, 0, len(r.extensions))
	for _, ext := range r.extensions {
		exts = append(exts, ext)
	}
	return exts
}

// GlobalRegistry is the global extension registry
var GlobalRegistry = NewExtensionRegistry()
