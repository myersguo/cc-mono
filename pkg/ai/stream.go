package ai

import (
	"context"

	"github.com/myersguo/cc-mono/pkg/ai/utils"
)

// AssistantMessageEventStream is a specialized event stream for assistant messages
type AssistantMessageEventStream struct {
	*utils.EventStream[AssistantMessageEvent, AssistantMessage]
}

// NewAssistantMessageEventStream creates a new assistant message event stream
func NewAssistantMessageEventStream(ctx context.Context) *AssistantMessageEventStream {
	return &AssistantMessageEventStream{
		EventStream: utils.NewEventStream[AssistantMessageEvent, AssistantMessage](ctx, 100),
	}
}

// Provider is the interface that all LLM providers must implement
type Provider interface {
	// Name returns the name of the provider
	Name() string

	// Stream sends a request to the LLM and returns a stream of events
	// This is the main method for tool-enabled interactions
	Stream(
		ctx context.Context,
		model Model,
		context Context,
		options *StreamOptions,
	) *AssistantMessageEventStream

	// StreamSimple sends a simple request without tools
	// This is a simplified version for basic text generation
	StreamSimple(
		ctx context.Context,
		model Model,
		context Context,
		options *SimpleStreamOptions,
	) *AssistantMessageEventStream

	// ValidateModel checks if the model is supported by this provider
	ValidateModel(model Model) error

	// GetDefaultModel returns the default model for this provider
	GetDefaultModel() Model
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name         string
	defaultModel Model
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, defaultModel Model) *BaseProvider {
	return &BaseProvider{
		name:         name,
		defaultModel: defaultModel,
	}
}

// Name returns the provider name
func (p *BaseProvider) Name() string {
	return p.name
}

// GetDefaultModel returns the default model
func (p *BaseProvider) GetDefaultModel() Model {
	return p.defaultModel
}
