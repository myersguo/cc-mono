package ai

import (
	"context"
	"testing"
)

func TestAssistantMessageEventStream(t *testing.T) {
	ctx := context.Background()
	stream := NewAssistantMessageEventStream(ctx)

	// Send some events
	go func() {
		stream.SendEvent(NewStartEvent())
		stream.SendEvent(NewTextDeltaEvent("Hello"))
		stream.SendEvent(NewTextDeltaEvent(" World"))
		stream.SendEvent(NewUsageEvent(Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}))
		stream.SendEvent(NewEndEvent(StopReasonEndTurn))

		// Send result
		result := NewAssistantMessage(
			[]Content{NewTextContent("Hello World")},
			"test-api",
			"test-provider",
			"test-model",
			Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15},
			StopReasonEndTurn,
		)
		stream.SendResult(result)
	}()

	// Consume events
	eventCount := 0
	for range stream.Events() {
		eventCount++
	}

	if eventCount != 5 {
		t.Errorf("Expected 5 events, got %d", eventCount)
	}

	// Get result
	result := <-stream.Result()
	if result.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", result.Usage.TotalTokens)
	}
}
