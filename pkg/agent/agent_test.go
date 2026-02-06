package agent

import (
	"context"
	"testing"
	"time"

	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/myersguo/cc-mono/pkg/ai/providers/openai"
)

func TestAgent(t *testing.T) {
	t.Run("CreateAgent", func(t *testing.T) {
		provider, _ := openai.NewProvider(openai.Config{
			APIKey: "test-key",
		})

		model := ai.Model{ID: "gpt-4", Provider: "openai"}
		agent := NewAgent(provider, "You are a helpful assistant", model, []AgentTool{})

		if agent == nil {
			t.Fatal("Expected agent to be created")
		}

		state := agent.GetState()
		if state.GetSystemPrompt() != "You are a helpful assistant" {
			t.Errorf("Expected system prompt 'You are a helpful assistant', got '%s'", state.GetSystemPrompt())
		}

		if state.GetModel().ID != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%s'", state.GetModel().ID)
		}
	})

	t.Run("AddTool", func(t *testing.T) {
		provider, _ := openai.NewProvider(openai.Config{
			APIKey: "test-key",
		})

		model := ai.Model{ID: "gpt-4", Provider: "openai"}
		agent := NewAgent(provider, "", model, []AgentTool{})

		tool := ai.NewTool("test_tool", "A test tool", map[string]any{})
		agentTool := NewAgentTool(tool, "Test", func(ctx context.Context, id string, params map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
			return AgentToolResult{}, nil
		})

		agent.AddTool(agentTool)

		foundTool, found := agent.FindTool("test_tool")
		if !found {
			t.Fatal("Expected to find tool 'test_tool'")
		}

		if foundTool.Tool.Name != "test_tool" {
			t.Errorf("Expected tool name 'test_tool', got '%s'", foundTool.Tool.Name)
		}
	})

	t.Run("EventBus", func(t *testing.T) {
		provider, _ := openai.NewProvider(openai.Config{
			APIKey: "test-key",
		})

		model := ai.Model{ID: "gpt-4", Provider: "openai"}
		agent := NewAgent(provider, "", model, []AgentTool{})

		eventBus := agent.GetEventBus()
		if eventBus == nil {
			t.Fatal("Expected event bus to exist")
		}

		// Subscribe to events
		events := eventBus.Subscribe(10)

		// Publish event
		eventBus.Publish(NewAgentStartEvent())

		// Receive event
		select {
		case event := <-events:
			if event.EventType() != EventTypeAgentStart {
				t.Errorf("Expected %s, got %s", EventTypeAgentStart, event.EventType())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for event")
		}

		agent.Close()
	})

	t.Run("AgentContext", func(t *testing.T) {
		provider, _ := openai.NewProvider(openai.Config{
			APIKey: "test-key",
		})

		model := ai.Model{ID: "gpt-4", Provider: "openai"}
		agent := NewAgent(provider, "", model, []AgentTool{})

		ctx := NewAgentContext(agent)

		if ctx.Agent != agent {
			t.Error("Expected agent to be set in context")
		}

		// Test steering queue
		msg := NewAgentMessage(ai.NewUserTextMessage("Steering"), "1", time.Now().UnixMilli())
		ctx.AddSteeringMessage(msg)

		if !ctx.HasSteeringMessages() {
			t.Error("Expected to have steering messages")
		}

		// Test follow-up queue
		followUp := NewAgentMessage(ai.NewUserTextMessage("Follow up"), "2", time.Now().UnixMilli())
		ctx.AddFollowUpMessage(followUp)

		if !ctx.HasFollowUpMessages() {
			t.Error("Expected to have follow-up messages")
		}
	})
}

func TestConvertMessagesToAI(t *testing.T) {
	agentMessages := []AgentMessage{
		NewAgentMessage(ai.NewUserTextMessage("Hello"), "1", time.Now().UnixMilli()),
		NewAgentMessage(ai.NewUserTextMessage("World"), "2", time.Now().UnixMilli()),
	}

	aiMessages := ConvertMessagesToAI(agentMessages)

	if len(aiMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(aiMessages))
	}

	// Check that messages are correctly converted
	userMsg1, ok := aiMessages[0].(ai.UserMessage)
	if !ok {
		t.Fatal("Expected UserMessage")
	}

	if len(userMsg1.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(userMsg1.Content))
	}
}

func TestBuildContext(t *testing.T) {
	model := ai.Model{ID: "test-model", Provider: "test"}
	state := NewAgentState("System prompt", model, []AgentTool{})

	messages := []AgentMessage{
		NewAgentMessage(ai.NewUserTextMessage("Hello"), "1", time.Now().UnixMilli()),
	}

	context := BuildContext(state, messages)

	if context.SystemPrompt != "System prompt" {
		t.Errorf("Expected 'System prompt', got '%s'", context.SystemPrompt)
	}

	if len(context.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(context.Messages))
	}
}

func TestBuildStreamOptions(t *testing.T) {
	tool := ai.NewTool("test", "Test", map[string]any{})
	agentTool := NewAgentTool(tool, "Test", func(ctx context.Context, id string, params map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error) {
		return AgentToolResult{}, nil
	})

	model := ai.Model{ID: "test-model", Provider: "test"}
	state := NewAgentState("", model, []AgentTool{agentTool})
	state.SetThinkingLevel(ThinkingLevelHigh)

	options := BuildStreamOptions(state)

	if len(options.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(options.Tools))
	}

	if options.ThinkingLevel != ai.ThinkingLevel(ThinkingLevelHigh) {
		t.Errorf("Expected ThinkingLevelHigh, got %s", options.ThinkingLevel)
	}
}
