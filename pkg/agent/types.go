package agent

import (
	"context"
	"sync"

	"github.com/myersguo/cc-mono/pkg/ai"
)

// ThinkingLevel represents the level of thinking/reasoning
type ThinkingLevel string

const (
	ThinkingLevelNone   ThinkingLevel = "none"
	ThinkingLevelLow    ThinkingLevel = "low"
	ThinkingLevelMedium ThinkingLevel = "medium"
	ThinkingLevelHigh   ThinkingLevel = "high"
)

// AgentMessage wraps ai.Message with additional metadata
type AgentMessage struct {
	Message   ai.Message `json:"message"`
	ID        string     `json:"id"`
	CreatedAt int64      `json:"created_at"`
}

// NewAgentMessage creates a new agent message
func NewAgentMessage(message ai.Message, id string, createdAt int64) AgentMessage {
	return AgentMessage{
		Message:   message,
		ID:        id,
		CreatedAt: createdAt,
	}
}

// AgentToolUpdateCallback is called when a tool has an update to report
type AgentToolUpdateCallback func(update AgentToolUpdate)

// AgentToolUpdate represents an update from a tool execution
type AgentToolUpdate struct {
	Type    string `json:"type"`    // "progress", "log", "error"
	Message string `json:"message"` // Update message
	Data    any    `json:"data,omitempty"`
}

// AgentToolResult represents the result of a tool execution
type AgentToolResult struct {
	Content []ai.Content `json:"content"`
	Details any          `json:"details,omitempty"`
	IsError bool         `json:"is_error"`
}

// AgentTool represents a tool that can be called by the agent
type AgentTool struct {
	// Tool definition for the LLM
	Tool ai.Tool

	// Label for display purposes
	Label string

	// Execute function that runs the tool
	Execute func(
		ctx context.Context,
		toolCallID string,
		params map[string]any,
		onUpdate AgentToolUpdateCallback,
	) (AgentToolResult, error)
}

// NewAgentTool creates a new agent tool
func NewAgentTool(
	tool ai.Tool,
	label string,
	execute func(context.Context, string, map[string]any, AgentToolUpdateCallback) (AgentToolResult, error),
) AgentTool {
	return AgentTool{
		Tool:    tool,
		Label:   label,
		Execute: execute,
	}
}

// AgentState represents the current state of the agent
type AgentState struct {
	mu sync.RWMutex

	// Configuration
	SystemPrompt  string
	Model         ai.Model
	ThinkingLevel ThinkingLevel

	// Tools available to the agent
	Tools []AgentTool

	// Message history
	Messages []AgentMessage

	// Streaming state
	IsStreaming   bool
	StreamMessage AgentMessage

	// Pending tool calls (tool call ID -> true)
	PendingToolCalls map[string]bool

	// Error state
	Error string
}

// NewAgentState creates a new agent state
func NewAgentState(systemPrompt string, model ai.Model, tools []AgentTool) *AgentState {
	return &AgentState{
		SystemPrompt:     systemPrompt,
		Model:            model,
		ThinkingLevel:    ThinkingLevelNone,
		Tools:            tools,
		Messages:         make([]AgentMessage, 0),
		PendingToolCalls: make(map[string]bool),
	}
}

// GetSystemPrompt returns the system prompt (thread-safe)
func (s *AgentState) GetSystemPrompt() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SystemPrompt
}

// SetSystemPrompt sets the system prompt (thread-safe)
func (s *AgentState) SetSystemPrompt(prompt string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SystemPrompt = prompt
}

// GetModel returns the model (thread-safe)
func (s *AgentState) GetModel() ai.Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Model
}

// SetModel sets the model (thread-safe)
func (s *AgentState) SetModel(model ai.Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = model
}

// GetThinkingLevel returns the thinking level (thread-safe)
func (s *AgentState) GetThinkingLevel() ThinkingLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ThinkingLevel
}

// SetThinkingLevel sets the thinking level (thread-safe)
func (s *AgentState) SetThinkingLevel(level ThinkingLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ThinkingLevel = level
}

// GetTools returns a copy of the tools (thread-safe)
func (s *AgentState) GetTools() []AgentTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tools := make([]AgentTool, len(s.Tools))
	copy(tools, s.Tools)
	return tools
}

// GetMessages returns a copy of the messages (thread-safe)
func (s *AgentState) GetMessages() []AgentMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	messages := make([]AgentMessage, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}

// AddMessage adds a message to the history (thread-safe)
func (s *AgentState) AddMessage(message AgentMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, message)
}

// SetIsStreaming sets the streaming state (thread-safe)
func (s *AgentState) SetIsStreaming(streaming bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsStreaming = streaming
}

// GetIsStreaming returns the streaming state (thread-safe)
func (s *AgentState) GetIsStreaming() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsStreaming
}

// SetStreamMessage sets the current streaming message (thread-safe)
func (s *AgentState) SetStreamMessage(message AgentMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StreamMessage = message
}

// GetStreamMessage returns the current streaming message (thread-safe)
func (s *AgentState) GetStreamMessage() AgentMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.StreamMessage
}

// AddPendingToolCall adds a pending tool call (thread-safe)
func (s *AgentState) AddPendingToolCall(toolCallID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PendingToolCalls[toolCallID] = true
}

// RemovePendingToolCall removes a pending tool call (thread-safe)
func (s *AgentState) RemovePendingToolCall(toolCallID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.PendingToolCalls, toolCallID)
}

// HasPendingToolCalls returns true if there are pending tool calls (thread-safe)
func (s *AgentState) HasPendingToolCalls() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.PendingToolCalls) > 0
}

// SetError sets the error state (thread-safe)
func (s *AgentState) SetError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
}

// GetError returns the error state (thread-safe)
func (s *AgentState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Error
}

// ClearError clears the error state (thread-safe)
func (s *AgentState) ClearError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = ""
}

// MessageQueue represents a queue for messages
type MessageQueue struct {
	mu       sync.Mutex
	messages []AgentMessage
}

// NewMessageQueue creates a new message queue
func NewMessageQueue() *MessageQueue {
	return &MessageQueue{
		messages: make([]AgentMessage, 0),
	}
}

// Push adds a message to the queue
func (q *MessageQueue) Push(message AgentMessage) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = append(q.messages, message)
}

// Pop removes and returns the first message from the queue
func (q *MessageQueue) Pop() (AgentMessage, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.messages) == 0 {
		return AgentMessage{}, false
	}

	message := q.messages[0]
	q.messages = q.messages[1:]
	return message, true
}

// Peek returns the first message without removing it
func (q *MessageQueue) Peek() (AgentMessage, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.messages) == 0 {
		return AgentMessage{}, false
	}

	return q.messages[0], true
}

// IsEmpty returns true if the queue is empty
func (q *MessageQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages) == 0
}

// Len returns the number of messages in the queue
func (q *MessageQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.messages)
}

// Clear removes all messages from the queue
func (q *MessageQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = make([]AgentMessage, 0)
}

// GetAll returns all messages in the queue without removing them
func (q *MessageQueue) GetAll() []AgentMessage {
	q.mu.Lock()
	defer q.mu.Unlock()
	messages := make([]AgentMessage, len(q.messages))
	copy(messages, q.messages)
	return messages
}
