package agent

import (
	"fmt"
	"testing"
	"time"
)

func TestEventBus(t *testing.T) {
	t.Run("BasicPubSub", func(t *testing.T) {
		bus := NewEventBus()
		defer bus.Close()

		// Subscribe
		ch := bus.Subscribe(10)

		// Publish event
		event := NewAgentStartEvent()
		bus.Publish(event)

		// Receive event
		select {
		case received := <-ch:
			if received.EventType() != EventTypeAgentStart {
				t.Errorf("Expected %s, got %s", EventTypeAgentStart, received.EventType())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for event")
		}
	})

	t.Run("MultipleSubscribers", func(t *testing.T) {
		bus := NewEventBus()
		defer bus.Close()

		// Create multiple subscribers
		ch1 := bus.Subscribe(10)
		ch2 := bus.Subscribe(10)
		ch3 := bus.Subscribe(10)

		if bus.ListenerCount() != 3 {
			t.Errorf("Expected 3 listeners, got %d", bus.ListenerCount())
		}

		// Publish event
		event := NewTurnStartEvent()
		bus.Publish(event)

		// All subscribers should receive the event
		for i, ch := range []<-chan AgentEvent{ch1, ch2, ch3} {
			select {
			case received := <-ch:
				if received.EventType() != EventTypeTurnStart {
					t.Errorf("Subscriber %d: Expected %s, got %s", i, EventTypeTurnStart, received.EventType())
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("Subscriber %d: Timeout waiting for event", i)
			}
		}
	})

	t.Run("CloseEventBus", func(t *testing.T) {
		bus := NewEventBus()

		ch := bus.Subscribe(10)

		bus.Close()

		if !bus.IsClosed() {
			t.Error("Expected bus to be closed")
		}

		// Channel should be closed
		select {
		case _, ok := <-ch:
			if ok {
				t.Error("Expected channel to be closed")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for channel close")
		}

		// Publishing after close should not panic
		bus.Publish(NewAgentStartEvent())
	})

	t.Run("NonBlockingSend", func(t *testing.T) {
		bus := NewEventBus()
		defer bus.Close()

		// Subscribe with small buffer
		ch := bus.Subscribe(1)

		// Fill the buffer
		bus.Publish(NewAgentStartEvent())

		// This should not block even though buffer is full
		bus.Publish(NewAgentStartEvent())
		bus.Publish(NewAgentStartEvent())

		// Drain the channel
		<-ch
	})
}

func TestEventTypes(t *testing.T) {
	t.Run("AgentStartEvent", func(t *testing.T) {
		event := NewAgentStartEvent()
		if event.EventType() != EventTypeAgentStart {
			t.Errorf("Expected %s, got %s", EventTypeAgentStart, event.EventType())
		}
	})

	t.Run("AgentEndEvent", func(t *testing.T) {
		messages := []AgentMessage{}
		event := NewAgentEndEvent(messages)
		if event.EventType() != EventTypeAgentEnd {
			t.Errorf("Expected %s, got %s", EventTypeAgentEnd, event.EventType())
		}
	})

	t.Run("TurnStartEvent", func(t *testing.T) {
		event := NewTurnStartEvent()
		if event.EventType() != EventTypeTurnStart {
			t.Errorf("Expected %s, got %s", EventTypeTurnStart, event.EventType())
		}
	})

	t.Run("ToolExecutionStartEvent", func(t *testing.T) {
		event := NewToolExecutionStartEvent("call-1", "test_tool", map[string]any{"arg": "value"})
		if event.EventType() != EventTypeToolExecutionStart {
			t.Errorf("Expected %s, got %s", EventTypeToolExecutionStart, event.EventType())
		}
		if event.ToolCallID != "call-1" {
			t.Errorf("Expected 'call-1', got '%s'", event.ToolCallID)
		}
		if event.ToolName != "test_tool" {
			t.Errorf("Expected 'test_tool', got '%s'", event.ToolName)
		}
	})

	t.Run("ErrorEvent", func(t *testing.T) {
		testErr := NewErrorEvent(fmt.Errorf("test error"), "test context")
		if testErr.EventType() != EventTypeError {
			t.Errorf("Expected %s, got %s", EventTypeError, testErr.EventType())
		}
		if testErr.Context != "test context" {
			t.Errorf("Expected 'test context', got '%s'", testErr.Context)
		}
		if testErr.Error != "test error" {
			t.Errorf("Expected 'test error', got '%s'", testErr.Error)
		}
	})
}
