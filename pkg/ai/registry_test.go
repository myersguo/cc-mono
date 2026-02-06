package ai

import (
	"context"
	"testing"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	*BaseProvider
}

func NewMockProvider(name string) *MockProvider {
	defaultModel := Model{
		ID:           "mock-model",
		Provider:     name,
		Name:         "Mock Model",
		ContextWindow: 100000,
	}
	return &MockProvider{
		BaseProvider: NewBaseProvider(name, defaultModel),
	}
}

func (p *MockProvider) Stream(
	ctx context.Context,
	model Model,
	context Context,
	options *StreamOptions,
) *AssistantMessageEventStream {
	stream := NewAssistantMessageEventStream(ctx)
	go func() {
		stream.SendEvent(NewStartEvent())
		stream.SendEvent(NewTextDeltaEvent("test"))
		stream.SendEvent(NewEndEvent(StopReasonEndTurn))
		result := NewAssistantMessage(
			[]Content{NewTextContent("test")},
			"mock",
			p.Name(),
			model.ID,
			Usage{},
			StopReasonEndTurn,
		)
		stream.SendResult(result)
	}()
	return stream
}

func (p *MockProvider) StreamSimple(
	ctx context.Context,
	model Model,
	context Context,
	options *SimpleStreamOptions,
) *AssistantMessageEventStream {
	return p.Stream(ctx, model, context, &StreamOptions{})
}

func (p *MockProvider) ValidateModel(model Model) error {
	return nil
}

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	t.Run("Register and Get", func(t *testing.T) {
		provider := NewMockProvider("test-provider")
		err := registry.Register(provider)
		if err != nil {
			t.Fatalf("Failed to register provider: %v", err)
		}

		retrieved, err := registry.Get("test-provider")
		if err != nil {
			t.Fatalf("Failed to get provider: %v", err)
		}

		if retrieved.Name() != "test-provider" {
			t.Errorf("Expected provider name 'test-provider', got %s", retrieved.Name())
		}
	})

	t.Run("Duplicate Registration", func(t *testing.T) {
		provider := NewMockProvider("duplicate")
		registry.Register(provider)

		err := registry.Register(provider)
		if err == nil {
			t.Error("Expected error when registering duplicate provider")
		}
	})

	t.Run("Get Nonexistent", func(t *testing.T) {
		_, err := registry.Get("nonexistent")
		if err == nil {
			t.Error("Expected error when getting nonexistent provider")
		}
	})

	t.Run("List", func(t *testing.T) {
		names := registry.List()
		if len(names) < 2 {
			t.Errorf("Expected at least 2 providers, got %d", len(names))
		}
	})

	t.Run("Unregister", func(t *testing.T) {
		provider := NewMockProvider("to-unregister")
		registry.Register(provider)

		err := registry.Unregister("to-unregister")
		if err != nil {
			t.Fatalf("Failed to unregister provider: %v", err)
		}

		_, err = registry.Get("to-unregister")
		if err == nil {
			t.Error("Expected error after unregistering provider")
		}
	})
}

func TestModelRegistry(t *testing.T) {
	registry := NewModelRegistry()

	t.Run("Register and Get", func(t *testing.T) {
		model := Model{
			ID:            "test-model",
			Provider:      "test-provider",
			Name:          "Test Model",
			ContextWindow: 100000,
		}

		err := registry.Register(model)
		if err != nil {
			t.Fatalf("Failed to register model: %v", err)
		}

		retrieved, err := registry.Get("test-model")
		if err != nil {
			t.Fatalf("Failed to get model: %v", err)
		}

		if retrieved.ID != "test-model" {
			t.Errorf("Expected model ID 'test-model', got %s", retrieved.ID)
		}
	})

	t.Run("Duplicate Registration", func(t *testing.T) {
		model := Model{ID: "duplicate-model", Provider: "test"}
		registry.Register(model)

		err := registry.Register(model)
		if err == nil {
			t.Error("Expected error when registering duplicate model")
		}
	})

	t.Run("List", func(t *testing.T) {
		models := registry.List()
		if len(models) < 2 {
			t.Errorf("Expected at least 2 models, got %d", len(models))
		}
	})

	t.Run("ListByProvider", func(t *testing.T) {
		model1 := Model{ID: "provider-a-1", Provider: "provider-a"}
		model2 := Model{ID: "provider-a-2", Provider: "provider-a"}
		model3 := Model{ID: "provider-b-1", Provider: "provider-b"}

		registry.Register(model1)
		registry.Register(model2)
		registry.Register(model3)

		models := registry.ListByProvider("provider-a")
		if len(models) != 2 {
			t.Errorf("Expected 2 models for provider-a, got %d", len(models))
		}
	})

	t.Run("Unregister", func(t *testing.T) {
		model := Model{ID: "to-unregister", Provider: "test"}
		registry.Register(model)

		err := registry.Unregister("to-unregister")
		if err != nil {
			t.Fatalf("Failed to unregister model: %v", err)
		}

		_, err = registry.Get("to-unregister")
		if err == nil {
			t.Error("Expected error after unregistering model")
		}
	})
}
