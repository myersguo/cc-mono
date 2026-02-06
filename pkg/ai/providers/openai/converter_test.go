package openai

import (
	"testing"

	"github.com/myersguo/cc-mono/pkg/ai"
)

func TestConvertUserMessage(t *testing.T) {
	t.Run("SimpleText", func(t *testing.T) {
		msg := ai.NewUserTextMessage("Hello, world!")
		result, err := convertUserMessage(msg)

		if err != nil {
			t.Fatalf("Failed to convert message: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(result))
		}

		if result[0].Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", result[0].Role)
		}

		if text, ok := result[0].Content.(string); ok {
			if text != "Hello, world!" {
				t.Errorf("Expected 'Hello, world!', got '%s'", text)
			}
		} else {
			t.Error("Expected string content")
		}
	})

	t.Run("MultimodalContent", func(t *testing.T) {
		msg := ai.NewUserMessage([]ai.Content{
			ai.NewTextContent("Check this image"),
			ai.NewImageContentFromURL("https://example.com/image.png", "image/png"),
		})

		result, err := convertUserMessage(msg)

		if err != nil {
			t.Fatalf("Failed to convert message: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(result))
		}

		// Content should be an array of parts
		if parts, ok := result[0].Content.([]ContentPart); ok {
			if len(parts) != 2 {
				t.Errorf("Expected 2 content parts, got %d", len(parts))
			}

			if parts[0].Type != "text" {
				t.Errorf("Expected text part, got '%s'", parts[0].Type)
			}

			if parts[1].Type != "image_url" {
				t.Errorf("Expected image_url part, got '%s'", parts[1].Type)
			}
		} else {
			t.Error("Expected content parts array")
		}
	})
}

func TestConvertAssistantMessage(t *testing.T) {
	t.Run("TextOnly", func(t *testing.T) {
		msg := ai.NewAssistantMessage(
			[]ai.Content{ai.NewTextContent("Hello!")},
			"test", "test", "test",
			ai.Usage{},
			ai.StopReasonEndTurn,
		)

		result, err := convertAssistantMessage(msg)

		if err != nil {
			t.Fatalf("Failed to convert message: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(result))
		}

		if result[0].Role != "assistant" {
			t.Errorf("Expected role 'assistant', got '%s'", result[0].Role)
		}

		if text, ok := result[0].Content.(string); ok {
			if text != "Hello!" {
				t.Errorf("Expected 'Hello!', got '%s'", text)
			}
		} else {
			t.Error("Expected string content")
		}
	})

	t.Run("WithToolCalls", func(t *testing.T) {
		toolCall := ai.NewToolCall("call-123", "read_file", map[string]any{
			"path": "test.txt",
		})

		msg := ai.NewAssistantMessage(
			[]ai.Content{toolCall},
			"test", "test", "test",
			ai.Usage{},
			ai.StopReasonToolUse,
		)

		result, err := convertAssistantMessage(msg)

		if err != nil {
			t.Fatalf("Failed to convert message: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(result))
		}

		if len(result[0].ToolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %d", len(result[0].ToolCalls))
		}

		tc := result[0].ToolCalls[0]
		if tc.ID != "call-123" {
			t.Errorf("Expected ID 'call-123', got '%s'", tc.ID)
		}

		if tc.Function.Name != "read_file" {
			t.Errorf("Expected name 'read_file', got '%s'", tc.Function.Name)
		}
	})
}

func TestConvertToolResultMessage(t *testing.T) {
	msg := ai.NewToolResultMessage(
		"call-123",
		"read_file",
		[]ai.Content{ai.NewTextContent("File content here")},
		false,
	)

	result, err := convertToolResultMessage(msg)

	if err != nil {
		t.Fatalf("Failed to convert message: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	if result[0].Role != "tool" {
		t.Errorf("Expected role 'tool', got '%s'", result[0].Role)
	}

	if result[0].ToolCallID != "call-123" {
		t.Errorf("Expected tool call ID 'call-123', got '%s'", result[0].ToolCallID)
	}

	if text, ok := result[0].Content.(string); ok {
		if text != "File content here" {
			t.Errorf("Expected 'File content here', got '%s'", text)
		}
	} else {
		t.Error("Expected string content")
	}
}

func TestConvertFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected ai.StopReason
	}{
		{"stop", ai.StopReasonEndTurn},
		{"length", ai.StopReasonMaxTokens},
		{"tool_calls", ai.StopReasonToolUse},
		{"content_filter", ai.StopReasonError},
		{"unknown", ai.StopReasonEndTurn},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertFinishReason(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestConvertContextToRequest(t *testing.T) {
	model := ai.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}

	context := ai.NewContext("You are a helpful assistant", []ai.Message{
		ai.NewUserTextMessage("Hello"),
	})

	options := &ai.StreamOptions{
		Tools: []ai.Tool{
			ai.NewTool("read_file", "Read a file", map[string]any{
				"type": "object",
			}),
		},
	}

	req, err := convertContextToRequest(model, context, options)

	if err != nil {
		t.Fatalf("Failed to convert context: %v", err)
	}

	if req.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", req.Model)
	}

	if !req.Stream {
		t.Error("Expected stream to be true")
	}

	// Should have system message + user message
	if len(req.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(req.Messages))
	}

	if req.Messages[0].Role != "system" {
		t.Errorf("Expected first message role 'system', got '%s'", req.Messages[0].Role)
	}

	if req.Messages[1].Role != "user" {
		t.Errorf("Expected second message role 'user', got '%s'", req.Messages[1].Role)
	}

	if len(req.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(req.Tools))
	}

	if req.Tools[0].Function.Name != "read_file" {
		t.Errorf("Expected tool name 'read_file', got '%s'", req.Tools[0].Function.Name)
	}
}
