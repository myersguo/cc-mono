package agent

import (
	"context"
	"testing"
	"time"

	"github.com/myersguo/cc-mono/pkg/ai"
)

func TestAgentState(t *testing.T) {
	model := ai.Model{ID: "test-model", Provider: "test"}
	tools := []AgentTool{}
	state := NewAgentState("Test prompt", model, tools)

	t.Run("GetSetSystemPrompt", func(t *testing.T) {
		prompt := state.GetSystemPrompt()
		if prompt != "Test prompt" {
			t.Errorf("Expected 'Test prompt', got '%s'", prompt)
		}

		state.SetSystemPrompt("New prompt")
		if state.GetSystemPrompt() != "New prompt" {
			t.Errorf("Expected 'New prompt', got '%s'", state.GetSystemPrompt())
		}
	})

	t.Run("GetSetModel", func(t *testing.T) {
		m := state.GetModel()
		if m.ID != "test-model" {
			t.Errorf("Expected 'test-model', got '%s'", m.ID)
		}

		newModel := ai.Model{ID: "new-model", Provider: "test"}
		state.SetModel(newModel)
		if state.GetModel().ID != "new-model" {
			t.Errorf("Expected 'new-model', got '%s'", state.GetModel().ID)
		}
	})

	t.Run("GetSetThinkingLevel", func(t *testing.T) {
		level := state.GetThinkingLevel()
		if level != ThinkingLevelNone {
			t.Errorf("Expected ThinkingLevelNone, got %s", level)
		}

		state.SetThinkingLevel(ThinkingLevelHigh)
		if state.GetThinkingLevel() != ThinkingLevelHigh {
			t.Errorf("Expected ThinkingLevelHigh, got %s", state.GetThinkingLevel())
		}
	})

	t.Run("Messages", func(t *testing.T) {
		msg := NewAgentMessage(
			ai.NewUserTextMessage("Hello"),
			"msg-1",
			time.Now().UnixMilli(),
		)

		state.AddMessage(msg)
		messages := state.GetMessages()

		if len(messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(messages))
		}

		if messages[0].ID != "msg-1" {
			t.Errorf("Expected message ID 'msg-1', got '%s'", messages[0].ID)
		}
	})

	t.Run("PendingToolCalls", func(t *testing.T) {
		state.AddPendingToolCall("call-1")
		state.AddPendingToolCall("call-2")

		if !state.HasPendingToolCalls() {
			t.Error("Expected to have pending tool calls")
		}

		state.RemovePendingToolCall("call-1")
		if !state.HasPendingToolCalls() {
			t.Error("Expected to still have pending tool calls")
		}

		state.RemovePendingToolCall("call-2")
		if state.HasPendingToolCalls() {
			t.Error("Expected no pending tool calls")
		}
	})

	t.Run("ErrorState", func(t *testing.T) {
		if state.GetError() != "" {
			t.Errorf("Expected no error, got '%s'", state.GetError())
		}

		state.SetError("test error")
		if state.GetError() != "test error" {
			t.Errorf("Expected 'test error', got '%s'", state.GetError())
		}

		state.ClearError()
		if state.GetError() != "" {
			t.Errorf("Expected no error after clear, got '%s'", state.GetError())
		}
	})
}

func TestMessageQueue(t *testing.T) {
	queue := NewMessageQueue()

	t.Run("EmptyQueue", func(t *testing.T) {
		if !queue.IsEmpty() {
			t.Error("Expected queue to be empty")
		}

		if queue.Len() != 0 {
			t.Errorf("Expected length 0, got %d", queue.Len())
		}

		_, ok := queue.Pop()
		if ok {
			t.Error("Expected Pop to return false on empty queue")
		}
	})

	t.Run("PushPop", func(t *testing.T) {
		msg1 := NewAgentMessage(ai.NewUserTextMessage("First"), "1", time.Now().UnixMilli())
		msg2 := NewAgentMessage(ai.NewUserTextMessage("Second"), "2", time.Now().UnixMilli())

		queue.Push(msg1)
		queue.Push(msg2)

		if queue.Len() != 2 {
			t.Errorf("Expected length 2, got %d", queue.Len())
		}

		popped, ok := queue.Pop()
		if !ok {
			t.Fatal("Expected Pop to succeed")
		}

		if popped.ID != "1" {
			t.Errorf("Expected first message, got ID '%s'", popped.ID)
		}

		if queue.Len() != 1 {
			t.Errorf("Expected length 1, got %d", queue.Len())
		}
	})

	t.Run("Peek", func(t *testing.T) {
		queue.Clear()
		msg := NewAgentMessage(ai.NewUserTextMessage("Test"), "test", time.Now().UnixMilli())
		queue.Push(msg)

		peeked, ok := queue.Peek()
		if !ok {
			t.Fatal("Expected Peek to succeed")
		}

		if peeked.ID != "test" {
			t.Errorf("Expected 'test', got '%s'", peeked.ID)
		}

		// Queue should still have the message
		if queue.Len() != 1 {
			t.Errorf("Expected length 1 after Peek, got %d", queue.Len())
		}
	})

	t.Run("Clear", func(t *testing.T) {
		queue.Push(NewAgentMessage(ai.NewUserTextMessage("1"), "1", time.Now().UnixMilli()))
		queue.Push(NewAgentMessage(ai.NewUserTextMessage("2"), "2", time.Now().UnixMilli()))

		queue.Clear()

		if !queue.IsEmpty() {
			t.Error("Expected queue to be empty after Clear")
		}
	})

	t.Run("GetAll", func(t *testing.T) {
		queue.Clear()
		msg1 := NewAgentMessage(ai.NewUserTextMessage("1"), "1", time.Now().UnixMilli())
		msg2 := NewAgentMessage(ai.NewUserTextMessage("2"), "2", time.Now().UnixMilli())

		queue.Push(msg1)
		queue.Push(msg2)

		all := queue.GetAll()
		if len(all) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(all))
		}

		// Queue should still have the messages
		if queue.Len() != 2 {
			t.Errorf("Expected length 2 after GetAll, got %d", queue.Len())
		}
	})
}

func TestAgentTool(t *testing.T) {
	tool := ai.NewTool("test_tool", "A test tool", map[string]any{
		"type": "object",
	})

	executed := false
	execute := func(ctx context.Context, id string, params map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
		executed = true
		return AgentToolResult{
			Content: []ai.Content{ai.NewTextContent("Result")},
			IsError: false,
		}, nil
	}

	agentTool := NewAgentTool(tool, "Test Tool", execute)

	if agentTool.Tool.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", agentTool.Tool.Name)
	}

	if agentTool.Label != "Test Tool" {
		t.Errorf("Expected label 'Test Tool', got '%s'", agentTool.Label)
	}

	// Test execution
	result, err := agentTool.Execute(nil, "call-1", map[string]any{}, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !executed {
		t.Error("Expected tool to be executed")
	}

	if result.IsError {
		t.Error("Expected result to not be an error")
	}
}
