package agent

import (
	"context"

	"github.com/myersguo/cc-mono/pkg/ai"
)

// Agent represents an AI agent that can interact with LLMs and execute tools
type Agent struct {
	state    *AgentState
	provider ai.Provider
	eventBus *EventBus
}

// NewAgent creates a new agent
func NewAgent(
	provider ai.Provider,
	systemPrompt string,
	model ai.Model,
	tools []AgentTool,
) *Agent {
	return &Agent{
		state:    NewAgentState(systemPrompt, model, tools),
		provider: provider,
		eventBus: NewEventBus(),
	}
}

// GetState returns the agent state (for reading)
func (a *Agent) GetState() *AgentState {
	return a.state
}

// GetProvider returns the provider
func (a *Agent) GetProvider() ai.Provider {
	return a.provider
}

// GetEventBus returns the event bus
func (a *Agent) GetEventBus() *EventBus {
	return a.eventBus
}

// SetSystemPrompt updates the system prompt
func (a *Agent) SetSystemPrompt(prompt string) {
	a.state.SetSystemPrompt(prompt)
}

// SetModel updates the model
func (a *Agent) SetModel(model ai.Model) {
	a.state.SetModel(model)
}

// SetThinkingLevel updates the thinking level
func (a *Agent) SetThinkingLevel(level ThinkingLevel) {
	a.state.SetThinkingLevel(level)
}

// AddTool adds a tool to the agent
func (a *Agent) AddTool(tool AgentTool) {
	a.state.mu.Lock()
	defer a.state.mu.Unlock()
	a.state.Tools = append(a.state.Tools, tool)
}

// FindTool finds a tool by name
func (a *Agent) FindTool(name string) (AgentTool, bool) {
	tools := a.state.GetTools()
	for _, tool := range tools {
		if tool.Tool.Name == name {
			return tool, true
		}
	}
	return AgentTool{}, false
}

// Run starts the agent with the given prompts
func (a *Agent) Run(ctx context.Context, prompts []AgentMessage) error {
	// Create default config
	config := &AgentLoopConfig{
		MaxTurns:       100,
		MaxToolCalls:   50,
		EnableSteering: true,
	}

	agentCtx := NewAgentContext(a)
	return AgentLoop(ctx, prompts, agentCtx, config, a.eventBus)
}

// RunWithConfig starts the agent with custom configuration
func (a *Agent) RunWithConfig(
	ctx context.Context,
	prompts []AgentMessage,
	config *AgentLoopConfig,
) error {
	agentCtx := NewAgentContext(a)
	return AgentLoop(ctx, prompts, agentCtx, config, a.eventBus)
}

// Close closes the agent and its event bus
func (a *Agent) Close() {
	a.eventBus.Close()
}

// AgentContext wraps the agent for use in the agent loop
type AgentContext struct {
	Agent         *Agent
	SteeringQueue *MessageQueue
	FollowUpQueue *MessageQueue
}

// NewAgentContext creates a new agent context
func NewAgentContext(agent *Agent) *AgentContext {
	return &AgentContext{
		Agent:         agent,
		SteeringQueue: NewMessageQueue(),
		FollowUpQueue: NewMessageQueue(),
	}
}

// AddSteeringMessage adds a steering message (interrupt current turn)
func (ctx *AgentContext) AddSteeringMessage(message AgentMessage) {
	ctx.SteeringQueue.Push(message)
}

// AddFollowUpMessage adds a follow-up message (after current turn)
func (ctx *AgentContext) AddFollowUpMessage(message AgentMessage) {
	ctx.FollowUpQueue.Push(message)
}

// HasSteeringMessages returns true if there are steering messages
func (ctx *AgentContext) HasSteeringMessages() bool {
	return !ctx.SteeringQueue.IsEmpty()
}

// HasFollowUpMessages returns true if there are follow-up messages
func (ctx *AgentContext) HasFollowUpMessages() bool {
	return !ctx.FollowUpQueue.IsEmpty()
}

// ConvertMessagesToAI converts agent messages to AI messages
func ConvertMessagesToAI(messages []AgentMessage) []ai.Message {
	aiMessages := make([]ai.Message, len(messages))
	for i, msg := range messages {
		aiMessages[i] = msg.Message
	}
	return aiMessages
}

// BuildContext builds an AI context from the agent state
func BuildContext(state *AgentState, messages []AgentMessage) ai.Context {
	return ai.NewContext(
		state.GetSystemPrompt(),
		ConvertMessagesToAI(messages),
	)
}

// BuildStreamOptions builds stream options from the agent state
func BuildStreamOptions(state *AgentState) *ai.StreamOptions {
	tools := state.GetTools()
	aiTools := make([]ai.Tool, len(tools))
	for i, tool := range tools {
		aiTools[i] = tool.Tool
	}

	return &ai.StreamOptions{
		Tools:         aiTools,
		ThinkingLevel: ai.ThinkingLevel(state.GetThinkingLevel()),
	}
}
