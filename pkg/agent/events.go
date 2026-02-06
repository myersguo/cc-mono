package agent

import (
	"github.com/myersguo/cc-mono/pkg/ai"
)

// AgentEventType represents the type of agent event
type AgentEventType string

const (
	// Agent lifecycle events
	EventTypeAgentStart AgentEventType = "agent_start"
	EventTypeAgentEnd   AgentEventType = "agent_end"

	// Turn events
	EventTypeTurnStart AgentEventType = "turn_start"
	EventTypeTurnEnd   AgentEventType = "turn_end"

	// Message events
	EventTypeMessageUpdate AgentEventType = "message_update"

	// Tool execution events
	EventTypeToolExecutionStart AgentEventType = "tool_execution_start"
	EventTypeToolExecutionEnd   AgentEventType = "tool_execution_end"

	// Permission events
	EventTypePermissionRequest AgentEventType = "permission_request"

	// Error events
	EventTypeError AgentEventType = "error"

	// Compaction events
	EventTypeCompactionStart AgentEventType = "compaction_start"
	EventTypeCompactionEnd   AgentEventType = "compaction_end"
)

// AgentEvent is the interface that all agent events implement
type AgentEvent interface {
	EventType() AgentEventType
	isAgentEvent() // private method to restrict implementation
}

// AgentStartEvent is emitted when the agent starts
type AgentStartEvent struct {
	Type AgentEventType `json:"type"`
}

func (e AgentStartEvent) EventType() AgentEventType { return e.Type }
func (e AgentStartEvent) isAgentEvent()             {}

// NewAgentStartEvent creates a new agent start event
func NewAgentStartEvent() AgentStartEvent {
	return AgentStartEvent{Type: EventTypeAgentStart}
}

// AgentEndEvent is emitted when the agent ends
type AgentEndEvent struct {
	Type     AgentEventType `json:"type"`
	Messages []AgentMessage `json:"messages"`
}

func (e AgentEndEvent) EventType() AgentEventType { return e.Type }
func (e AgentEndEvent) isAgentEvent()             {}

// NewAgentEndEvent creates a new agent end event
func NewAgentEndEvent(messages []AgentMessage) AgentEndEvent {
	return AgentEndEvent{
		Type:     EventTypeAgentEnd,
		Messages: messages,
	}
}

// TurnStartEvent is emitted when a new turn starts
type TurnStartEvent struct {
	Type AgentEventType `json:"type"`
}

func (e TurnStartEvent) EventType() AgentEventType { return e.Type }
func (e TurnStartEvent) isAgentEvent()             {}

// NewTurnStartEvent creates a new turn start event
func NewTurnStartEvent() TurnStartEvent {
	return TurnStartEvent{Type: EventTypeTurnStart}
}

// TurnEndEvent is emitted when a turn ends
type TurnEndEvent struct {
	Type        AgentEventType         `json:"type"`
	Message     AgentMessage           `json:"message"`
	ToolResults []ai.ToolResultMessage `json:"tool_results,omitempty"`
}

func (e TurnEndEvent) EventType() AgentEventType { return e.Type }
func (e TurnEndEvent) isAgentEvent()             {}

// NewTurnEndEvent creates a new turn end event
func NewTurnEndEvent(message AgentMessage, toolResults []ai.ToolResultMessage) TurnEndEvent {
	return TurnEndEvent{
		Type:        EventTypeTurnEnd,
		Message:     message,
		ToolResults: toolResults,
	}
}

// MessageUpdateEvent is emitted when a message is updated (during streaming)
type MessageUpdateEvent struct {
	Type                  AgentEventType            `json:"type"`
	Message               AgentMessage              `json:"message"`
	AssistantMessageEvent ai.AssistantMessageEvent  `json:"assistant_message_event"`
}

func (e MessageUpdateEvent) EventType() AgentEventType { return e.Type }
func (e MessageUpdateEvent) isAgentEvent()             {}

// NewMessageUpdateEvent creates a new message update event
func NewMessageUpdateEvent(message AgentMessage, assistantEvent ai.AssistantMessageEvent) MessageUpdateEvent {
	return MessageUpdateEvent{
		Type:                  EventTypeMessageUpdate,
		Message:               message,
		AssistantMessageEvent: assistantEvent,
	}
}

// ToolExecutionStartEvent is emitted when a tool execution starts
type ToolExecutionStartEvent struct {
	Type       AgentEventType `json:"type"`
	ToolCallID string         `json:"tool_call_id"`
	ToolName   string         `json:"tool_name"`
	Args       map[string]any `json:"args"`
}

func (e ToolExecutionStartEvent) EventType() AgentEventType { return e.Type }
func (e ToolExecutionStartEvent) isAgentEvent()             {}

// NewToolExecutionStartEvent creates a new tool execution start event
func NewToolExecutionStartEvent(toolCallID, toolName string, args map[string]any) ToolExecutionStartEvent {
	return ToolExecutionStartEvent{
		Type:       EventTypeToolExecutionStart,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Args:       args,
	}
}

// ToolExecutionEndEvent is emitted when a tool execution ends
type ToolExecutionEndEvent struct {
	Type       AgentEventType `json:"type"`
	ToolCallID string         `json:"tool_call_id"`
	ToolName   string         `json:"tool_name"`
	Result     any            `json:"result"`
	IsError    bool           `json:"is_error"`
}

func (e ToolExecutionEndEvent) EventType() AgentEventType { return e.Type }
func (e ToolExecutionEndEvent) isAgentEvent()             {}

// NewToolExecutionEndEvent creates a new tool execution end event
func NewToolExecutionEndEvent(toolCallID, toolName string, result any, isError bool) ToolExecutionEndEvent {
	return ToolExecutionEndEvent{
		Type:       EventTypeToolExecutionEnd,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Result:     result,
		IsError:    isError,
	}
}

// ErrorEvent is emitted when an error occurs
type ErrorEvent struct {
	Type    AgentEventType `json:"type"`
	Error   string         `json:"error"`
	Context string         `json:"context,omitempty"`
}

func (e ErrorEvent) EventType() AgentEventType { return e.Type }
func (e ErrorEvent) isAgentEvent()             {}

// NewErrorEvent creates a new error event
func NewErrorEvent(err error, context string) ErrorEvent {
	return ErrorEvent{
		Type:    EventTypeError,
		Error:   err.Error(),
		Context: context,
	}
}

// CompactionStartEvent is emitted when context compaction starts
type CompactionStartEvent struct {
	Type         AgentEventType `json:"type"`
	MessageCount int            `json:"message_count"`
	TokenCount   int            `json:"token_count,omitempty"`
}

func (e CompactionStartEvent) EventType() AgentEventType { return e.Type }
func (e CompactionStartEvent) isAgentEvent()             {}

// NewCompactionStartEvent creates a new compaction start event
func NewCompactionStartEvent(messageCount, tokenCount int) CompactionStartEvent {
	return CompactionStartEvent{
		Type:         EventTypeCompactionStart,
		MessageCount: messageCount,
		TokenCount:   tokenCount,
	}
}

// CompactionEndEvent is emitted when context compaction ends
type CompactionEndEvent struct {
	Type         AgentEventType `json:"type"`
	MessageCount int            `json:"message_count"`
	TokenCount   int            `json:"token_count,omitempty"`
}

func (e CompactionEndEvent) EventType() AgentEventType { return e.Type }
func (e CompactionEndEvent) isAgentEvent()             {}

// NewCompactionEndEvent creates a new compaction end event
func NewCompactionEndEvent(messageCount, tokenCount int) CompactionEndEvent {
	return CompactionEndEvent{
		Type:         EventTypeCompactionEnd,
		MessageCount: messageCount,
		TokenCount:   tokenCount,
	}
}

// PermissionRequestEvent is emitted when a tool needs user permission
type PermissionRequestEvent struct {
	Type    AgentEventType      `json:"type"`
	Request *PermissionRequest  `json:"request"`
}

func (e PermissionRequestEvent) EventType() AgentEventType { return e.Type }
func (e PermissionRequestEvent) isAgentEvent()             {}

// NewPermissionRequestEvent creates a new permission request event
func NewPermissionRequestEvent(request *PermissionRequest) PermissionRequestEvent {
	return PermissionRequestEvent{
		Type:    EventTypePermissionRequest,
		Request: request,
	}
}
