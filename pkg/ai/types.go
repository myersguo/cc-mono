package ai

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeUser       MessageType = "user"
	MessageTypeAssistant  MessageType = "assistant"
	MessageTypeToolResult MessageType = "tool_result"
)

// Message is the interface that all message types implement
// Discriminated union pattern using private method
type Message interface {
	GetType() MessageType
	GetTimestamp() int64
	isMessage() // private method to restrict implementation
}

// UserMessage represents a message from the user
type UserMessage struct {
	Type      MessageType `json:"type"`
	Content   []Content   `json:"content"`
	Timestamp int64       `json:"timestamp"`
}

func (m UserMessage) GetType() MessageType   { return m.Type }
func (m UserMessage) GetTimestamp() int64    { return m.Timestamp }
func (m UserMessage) isMessage()             {}

// NewUserMessage creates a new user message
func NewUserMessage(content []Content) UserMessage {
	return UserMessage{
		Type:      MessageTypeUser,
		Content:   content,
		Timestamp: time.Now().UnixMilli(),
	}
}

// NewUserTextMessage creates a new user message with text content
func NewUserTextMessage(text string) UserMessage {
	return NewUserMessage([]Content{TextContent{Type: ContentTypeText, Text: text}})
}

// MarshalJSON implements json.Marshaler for UserMessage
func (m UserMessage) MarshalJSON() ([]byte, error) {
	type Alias UserMessage
	return json.Marshal(&struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Content: marshalContents(m.Content),
		Alias:   (*Alias)(&m),
	})
}

// UnmarshalJSON implements json.Unmarshaler for UserMessage
func (m *UserMessage) UnmarshalJSON(data []byte) error {
	type Alias UserMessage
	aux := &struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	contents, err := unmarshalContents(aux.Content)
	if err != nil {
		return err
	}
	m.Content = contents

	return nil
}

// StopReason represents why the model stopped generating
type StopReason string

const (
	StopReasonEndTurn      StopReason = "end_turn"
	StopReasonMaxTokens    StopReason = "max_tokens"
	StopReasonToolUse      StopReason = "tool_use"
	StopReasonStopSequence StopReason = "stop_sequence"
	StopReasonError        StopReason = "error"
)

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// AssistantMessage represents a message from the assistant (LLM)
type AssistantMessage struct {
	Type         MessageType `json:"type"`
	Content      []Content   `json:"content"`
	API          string      `json:"api"`
	Provider     string      `json:"provider"`
	Model        string      `json:"model"`
	Usage        Usage       `json:"usage"`
	StopReason   StopReason  `json:"stop_reason"`
	ErrorMessage string      `json:"error_message,omitempty"`
	Timestamp    int64       `json:"timestamp"`
}

func (m AssistantMessage) GetType() MessageType { return m.Type }
func (m AssistantMessage) GetTimestamp() int64  { return m.Timestamp }
func (m AssistantMessage) isMessage()           {}

// NewAssistantMessage creates a new assistant message
func NewAssistantMessage(
	content []Content,
	api, provider, model string,
	usage Usage,
	stopReason StopReason,
) AssistantMessage {
	return AssistantMessage{
		Type:       MessageTypeAssistant,
		Content:    content,
		API:        api,
		Provider:   provider,
		Model:      model,
		Usage:      usage,
		StopReason: stopReason,
		Timestamp:  time.Now().UnixMilli(),
	}
}

// MarshalJSON implements json.Marshaler for AssistantMessage
func (m AssistantMessage) MarshalJSON() ([]byte, error) {
	type Alias AssistantMessage
	return json.Marshal(&struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Content: marshalContents(m.Content),
		Alias:   (*Alias)(&m),
	})
}

// UnmarshalJSON implements json.Unmarshaler for AssistantMessage
func (m *AssistantMessage) UnmarshalJSON(data []byte) error {
	type Alias AssistantMessage
	aux := &struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	contents, err := unmarshalContents(aux.Content)
	if err != nil {
		return err
	}
	m.Content = contents

	return nil
}

// ToolResultMessage represents the result of a tool execution
type ToolResultMessage struct {
	Type       MessageType `json:"type"`
	ToolCallID string      `json:"tool_call_id"`
	ToolName   string      `json:"tool_name"`
	Content    []Content   `json:"content"`
	Details    any         `json:"details,omitempty"`
	IsError    bool        `json:"is_error"`
	Timestamp  int64       `json:"timestamp"`
}

func (m ToolResultMessage) GetType() MessageType { return m.Type }
func (m ToolResultMessage) GetTimestamp() int64  { return m.Timestamp }
func (m ToolResultMessage) isMessage()           {}

// NewToolResultMessage creates a new tool result message
func NewToolResultMessage(
	toolCallID, toolName string,
	content []Content,
	isError bool,
) ToolResultMessage {
	return ToolResultMessage{
		Type:       MessageTypeToolResult,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Content:    content,
		IsError:    isError,
		Timestamp:  time.Now().UnixMilli(),
	}
}

// MarshalJSON implements json.Marshaler for ToolResultMessage
func (m ToolResultMessage) MarshalJSON() ([]byte, error) {
	type Alias ToolResultMessage
	return json.Marshal(&struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Content: marshalContents(m.Content),
		Alias:   (*Alias)(&m),
	})
}

// UnmarshalJSON implements json.Unmarshaler for ToolResultMessage
func (m *ToolResultMessage) UnmarshalJSON(data []byte) error {
	type Alias ToolResultMessage
	aux := &struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	contents, err := unmarshalContents(aux.Content)
	if err != nil {
		return err
	}
	m.Content = contents

	return nil
}

// ContentType represents the type of content
type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeThinking ContentType = "thinking"
	ContentTypeToolCall ContentType = "tool_call"
	ContentTypeImage    ContentType = "image"
)

// Content is the interface that all content types implement
// Discriminated union pattern using private method
type Content interface {
	ContentType() ContentType
	isContent() // private method to restrict implementation
}

// TextContent represents text content
type TextContent struct {
	Type ContentType `json:"type"`
	Text string      `json:"text"`
}

func (c TextContent) ContentType() ContentType { return ContentTypeText }
func (c TextContent) isContent()               {}

// NewTextContent creates new text content
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: ContentTypeText,
		Text: text,
	}
}

// ThinkingContent represents thinking/reasoning content
type ThinkingContent struct {
	Type     ContentType `json:"type"`
	Thinking string      `json:"thinking"`
}

func (c ThinkingContent) ContentType() ContentType { return ContentTypeThinking }
func (c ThinkingContent) isContent()               {}

// NewThinkingContent creates new thinking content
func NewThinkingContent(thinking string) ThinkingContent {
	return ThinkingContent{
		Type:     ContentTypeThinking,
		Thinking: thinking,
	}
}

// ToolCall represents a tool call from the assistant
type ToolCall struct {
	Type   ContentType            `json:"type"`
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Params map[string]any `json:"params"`
}

func (c ToolCall) ContentType() ContentType { return ContentTypeToolCall }
func (c ToolCall) isContent()               {}

// NewToolCall creates a new tool call
func NewToolCall(id, name string, params map[string]any) ToolCall {
	return ToolCall{
		Type:   ContentTypeToolCall,
		ID:     id,
		Name:   name,
		Params: params,
	}
}

// ImageSource represents the source of an image
type ImageSource struct {
	Type      string `json:"type"`       // "url" or "base64"
	URL       string `json:"url,omitempty"`
	Data      string `json:"data,omitempty"`
	MediaType string `json:"media_type"` // e.g., "image/png"
}

// ImageContent represents image content
type ImageContent struct {
	Type   ContentType `json:"type"`
	Source ImageSource `json:"source"`
}

func (c ImageContent) ContentType() ContentType { return ContentTypeImage }
func (c ImageContent) isContent()               {}

// NewImageContentFromURL creates new image content from URL
func NewImageContentFromURL(url, mediaType string) ImageContent {
	return ImageContent{
		Type: ContentTypeImage,
		Source: ImageSource{
			Type:      "url",
			URL:       url,
			MediaType: mediaType,
		},
	}
}

// NewImageContentFromBase64 creates new image content from base64 data
func NewImageContentFromBase64(data, mediaType string) ImageContent {
	return ImageContent{
		Type: ContentTypeImage,
		Source: ImageSource{
			Type:      "base64",
			Data:      data,
			MediaType: mediaType,
		},
	}
}

// JSON marshaling and unmarshaling for polymorphic types

// MarshalJSON implements json.Marshaler for Message types
func MarshalMessage(m Message) ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalMessage unmarshals a message from JSON
func UnmarshalMessage(data []byte) (Message, error) {
	// First, unmarshal to get the type
	var raw struct {
		Type MessageType `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Then unmarshal to the appropriate concrete type
	switch raw.Type {
	case MessageTypeUser:
		var msg UserMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case MessageTypeAssistant:
		var msg AssistantMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	case MessageTypeToolResult:
		var msg ToolResultMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", raw.Type)
	}
}

// UnmarshalContent unmarshals content from JSON
func UnmarshalContent(data []byte) (Content, error) {
	// First, unmarshal to get the type
	var raw struct {
		Type ContentType `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// Then unmarshal to the appropriate concrete type
	switch raw.Type {
	case ContentTypeText:
		var content TextContent
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeThinking:
		var content ThinkingContent
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeToolCall:
		var content ToolCall
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, err
		}
		return content, nil
	case ContentTypeImage:
		var content ImageContent
		if err := json.Unmarshal(data, &content); err != nil {
			return nil, err
		}
		return content, nil
	default:
		return nil, fmt.Errorf("unknown content type: %s", raw.Type)
	}
}

// Helper functions for marshaling/unmarshaling content arrays
func marshalContents(contents []Content) []json.RawMessage {
	result := make([]json.RawMessage, len(contents))
	for i, content := range contents {
		data, _ := json.Marshal(content)
		result[i] = data
	}
	return result
}

func unmarshalContents(raw []json.RawMessage) ([]Content, error) {
	contents := make([]Content, len(raw))
	for i, data := range raw {
		content, err := UnmarshalContent(data)
		if err != nil {
			return nil, err
		}
		contents[i] = content
	}
	return contents, nil
}

// Context represents the context for an LLM request
type Context struct {
	SystemPrompt string    `json:"system_prompt"`
	Messages     []Message `json:"messages"`
}

// NewContext creates a new context
func NewContext(systemPrompt string, messages []Message) Context {
	return Context{
		SystemPrompt: systemPrompt,
		Messages:     messages,
	}
}

// MarshalJSON implements json.Marshaler for Context
func (c Context) MarshalJSON() ([]byte, error) {
	type Alias Context
	return json.Marshal(&struct {
		Messages []json.RawMessage `json:"messages"`
		*Alias
	}{
		Messages: marshalMessages(c.Messages),
		Alias:    (*Alias)(&c),
	})
}

// UnmarshalJSON implements json.Unmarshaler for Context
func (c *Context) UnmarshalJSON(data []byte) error {
	type Alias Context
	aux := &struct {
		Messages []json.RawMessage `json:"messages"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	messages, err := unmarshalMessages(aux.Messages)
	if err != nil {
		return err
	}
	c.Messages = messages

	return nil
}

func marshalMessages(messages []Message) []json.RawMessage {
	result := make([]json.RawMessage, len(messages))
	for i, msg := range messages {
		data, _ := MarshalMessage(msg)
		result[i] = data
	}
	return result
}

func unmarshalMessages(raw []json.RawMessage) ([]Message, error) {
	messages := make([]Message, len(raw))
	for i, data := range raw {
		msg, err := UnmarshalMessage(data)
		if err != nil {
			return nil, err
		}
		messages[i] = msg
	}
	return messages, nil
}

// Tool represents a tool that can be called by the LLM
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}

// NewTool creates a new tool definition
func NewTool(name, description string, parameters map[string]any) Tool {
	return Tool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
	}
}

// ThinkingLevel represents the level of thinking/reasoning
type ThinkingLevel string

const (
	ThinkingLevelNone   ThinkingLevel = "none"
	ThinkingLevelLow    ThinkingLevel = "low"
	ThinkingLevelMedium ThinkingLevel = "medium"
	ThinkingLevelHigh   ThinkingLevel = "high"
)

// Model represents an LLM model
type Model struct {
	ID              string        `json:"id"`
	Provider        string        `json:"provider"`
	Name            string        `json:"name"`
	ContextWindow   int           `json:"context_window"`
	MaxOutput       int           `json:"max_output"`
	InputCostPer1M  float64       `json:"input_cost_per_million"`
	OutputCostPer1M float64       `json:"output_cost_per_million"`
	SupportsVision  bool          `json:"supports_vision"`
	SupportsTools   bool          `json:"supports_tools"`
	SupportsThinking bool         `json:"supports_thinking,omitempty"`
	ThinkingLevel   ThinkingLevel `json:"thinking_level,omitempty"`
}

// CalculateCost calculates the cost for the given usage
func (m Model) CalculateCost(usage Usage) float64 {
	inputCost := float64(usage.InputTokens) * m.InputCostPer1M / 1000000.0
	outputCost := float64(usage.OutputTokens) * m.OutputCostPer1M / 1000000.0
	return inputCost + outputCost
}

// StreamOptions represents options for streaming
type StreamOptions struct {
	Tools         []Tool        `json:"tools,omitempty"`
	ThinkingLevel ThinkingLevel `json:"thinking_level,omitempty"`
	Temperature   *float64      `json:"temperature,omitempty"`
	MaxTokens     *int          `json:"max_tokens,omitempty"`
}

// SimpleStreamOptions represents simplified options for simple streaming
type SimpleStreamOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

// AssistantMessageEventType represents the type of assistant message event
type AssistantMessageEventType string

const (
	EventTypeStart       AssistantMessageEventType = "start"
	EventTypeContentDelta AssistantMessageEventType = "content_delta"
	EventTypeToolCall     AssistantMessageEventType = "tool_call"
	EventTypeUsage        AssistantMessageEventType = "usage"
	EventTypeEnd          AssistantMessageEventType = "end"
	EventTypeError        AssistantMessageEventType = "error"
)

// AssistantMessageEvent represents an event in the assistant message stream
type AssistantMessageEvent struct {
	Type AssistantMessageEventType `json:"type"`

	// For content_delta
	ContentType  ContentType `json:"content_type,omitempty"`
	TextDelta    string      `json:"text_delta,omitempty"`
	ThinkingDelta string     `json:"thinking_delta,omitempty"`

	// For tool_call
	ToolCall *ToolCall `json:"tool_call,omitempty"`

	// For usage
	Usage *Usage `json:"usage,omitempty"`

	// For end
	StopReason StopReason `json:"stop_reason,omitempty"`

	// For error
	Error string `json:"error,omitempty"`
}

// NewStartEvent creates a new start event
func NewStartEvent() AssistantMessageEvent {
	return AssistantMessageEvent{Type: EventTypeStart}
}

// NewTextDeltaEvent creates a new text delta event
func NewTextDeltaEvent(delta string) AssistantMessageEvent {
	return AssistantMessageEvent{
		Type:        EventTypeContentDelta,
		ContentType: ContentTypeText,
		TextDelta:   delta,
	}
}

// NewThinkingDeltaEvent creates a new thinking delta event
func NewThinkingDeltaEvent(delta string) AssistantMessageEvent {
	return AssistantMessageEvent{
		Type:          EventTypeContentDelta,
		ContentType:   ContentTypeThinking,
		ThinkingDelta: delta,
	}
}

// NewToolCallEvent creates a new tool call event
func NewToolCallEvent(toolCall ToolCall) AssistantMessageEvent {
	return AssistantMessageEvent{
		Type:     EventTypeToolCall,
		ToolCall: &toolCall,
	}
}

// NewUsageEvent creates a new usage event
func NewUsageEvent(usage Usage) AssistantMessageEvent {
	return AssistantMessageEvent{
		Type:  EventTypeUsage,
		Usage: &usage,
	}
}

// NewEndEvent creates a new end event
func NewEndEvent(stopReason StopReason) AssistantMessageEvent {
	return AssistantMessageEvent{
		Type:       EventTypeEnd,
		StopReason: stopReason,
	}
}

// NewErrorEvent creates a new error event
func NewErrorEvent(err error) AssistantMessageEvent {
	return AssistantMessageEvent{
		Type:  EventTypeError,
		Error: err.Error(),
	}
}
