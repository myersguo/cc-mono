package ai

import (
	"encoding/json"
	"testing"
)

func TestMessageTypes(t *testing.T) {
	t.Run("UserMessage", func(t *testing.T) {
		msg := NewUserTextMessage("Hello, world!")
		if msg.GetType() != MessageTypeUser {
			t.Errorf("Expected type %s, got %s", MessageTypeUser, msg.GetType())
		}
		if len(msg.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(msg.Content))
		}
	})

	t.Run("AssistantMessage", func(t *testing.T) {
		content := []Content{NewTextContent("Response")}
		usage := Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30}
		msg := NewAssistantMessage(content, "api", "provider", "model", usage, StopReasonEndTurn)

		if msg.GetType() != MessageTypeAssistant {
			t.Errorf("Expected type %s, got %s", MessageTypeAssistant, msg.GetType())
		}
		if msg.Usage.TotalTokens != 30 {
			t.Errorf("Expected 30 total tokens, got %d", msg.Usage.TotalTokens)
		}
	})

	t.Run("ToolResultMessage", func(t *testing.T) {
		content := []Content{NewTextContent("Result")}
		msg := NewToolResultMessage("call-123", "read", content, false)

		if msg.GetType() != MessageTypeToolResult {
			t.Errorf("Expected type %s, got %s", MessageTypeToolResult, msg.GetType())
		}
		if msg.ToolCallID != "call-123" {
			t.Errorf("Expected tool call ID 'call-123', got %s", msg.ToolCallID)
		}
	})
}

func TestContentTypes(t *testing.T) {
	t.Run("TextContent", func(t *testing.T) {
		content := NewTextContent("Hello")
		if content.ContentType() != ContentTypeText {
			t.Errorf("Expected type %s, got %s", ContentTypeText, content.ContentType())
		}
		if content.Text != "Hello" {
			t.Errorf("Expected text 'Hello', got %s", content.Text)
		}
	})

	t.Run("ThinkingContent", func(t *testing.T) {
		content := NewThinkingContent("I think...")
		if content.ContentType() != ContentTypeThinking {
			t.Errorf("Expected type %s, got %s", ContentTypeThinking, content.ContentType())
		}
	})

	t.Run("ToolCall", func(t *testing.T) {
		params := map[string]any{"file": "test.txt"}
		toolCall := NewToolCall("call-123", "read", params)
		if toolCall.ContentType() != ContentTypeToolCall {
			t.Errorf("Expected type %s, got %s", ContentTypeToolCall, toolCall.ContentType())
		}
		if toolCall.Name != "read" {
			t.Errorf("Expected name 'read', got %s", toolCall.Name)
		}
	})

	t.Run("ImageContent", func(t *testing.T) {
		content := NewImageContentFromURL("http://example.com/image.png", "image/png")
		if content.ContentType() != ContentTypeImage {
			t.Errorf("Expected type %s, got %s", ContentTypeImage, content.ContentType())
		}
		if content.Source.Type != "url" {
			t.Errorf("Expected source type 'url', got %s", content.Source.Type)
		}
	})
}

func TestJSONMarshaling(t *testing.T) {
	t.Run("UserMessage JSON", func(t *testing.T) {
		msg := NewUserTextMessage("Hello")
		data, err := MarshalMessage(msg)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		unmarshaled, err := UnmarshalMessage(data)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		userMsg, ok := unmarshaled.(UserMessage)
		if !ok {
			t.Fatal("Expected UserMessage")
		}
		if len(userMsg.Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(userMsg.Content))
		}
	})

	t.Run("Context JSON", func(t *testing.T) {
		messages := []Message{
			NewUserTextMessage("Hello"),
		}
		ctx := NewContext("System prompt", messages)

		data, err := json.Marshal(ctx)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var unmarshaled Context
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.SystemPrompt != "System prompt" {
			t.Errorf("Expected system prompt 'System prompt', got %s", unmarshaled.SystemPrompt)
		}
		if len(unmarshaled.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(unmarshaled.Messages))
		}
	})
}

func TestModelCostCalculation(t *testing.T) {
	model := Model{
		ID:              "test-model",
		InputCostPer1M:  1.0,
		OutputCostPer1M: 2.0,
	}

	usage := Usage{
		InputTokens:  1000,
		OutputTokens: 500,
		TotalTokens:  1500,
	}

	cost := model.CalculateCost(usage)
	expected := (1000.0 * 1.0 / 1000000.0) + (500.0 * 2.0 / 1000000.0)

	if cost != expected {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestAssistantMessageEvents(t *testing.T) {
	t.Run("StartEvent", func(t *testing.T) {
		event := NewStartEvent()
		if event.Type != EventTypeStart {
			t.Errorf("Expected type %s, got %s", EventTypeStart, event.Type)
		}
	})

	t.Run("TextDeltaEvent", func(t *testing.T) {
		event := NewTextDeltaEvent("Hello")
		if event.Type != EventTypeContentDelta {
			t.Errorf("Expected type %s, got %s", EventTypeContentDelta, event.Type)
		}
		if event.TextDelta != "Hello" {
			t.Errorf("Expected delta 'Hello', got %s", event.TextDelta)
		}
	})

	t.Run("ToolCallEvent", func(t *testing.T) {
		toolCall := NewToolCall("call-123", "read", map[string]any{"file": "test.txt"})
		event := NewToolCallEvent(toolCall)
		if event.Type != EventTypeToolCall {
			t.Errorf("Expected type %s, got %s", EventTypeToolCall, event.Type)
		}
		if event.ToolCall == nil {
			t.Fatal("Expected tool call to be set")
		}
		if event.ToolCall.Name != "read" {
			t.Errorf("Expected tool call name 'read', got %s", event.ToolCall.Name)
		}
	})

	t.Run("UsageEvent", func(t *testing.T) {
		usage := Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30}
		event := NewUsageEvent(usage)
		if event.Type != EventTypeUsage {
			t.Errorf("Expected type %s, got %s", EventTypeUsage, event.Type)
		}
		if event.Usage.TotalTokens != 30 {
			t.Errorf("Expected 30 total tokens, got %d", event.Usage.TotalTokens)
		}
	})

	t.Run("EndEvent", func(t *testing.T) {
		event := NewEndEvent(StopReasonEndTurn)
		if event.Type != EventTypeEnd {
			t.Errorf("Expected type %s, got %s", EventTypeEnd, event.Type)
		}
		if event.StopReason != StopReasonEndTurn {
			t.Errorf("Expected stop reason %s, got %s", StopReasonEndTurn, event.StopReason)
		}
	})
}
