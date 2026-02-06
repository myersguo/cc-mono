package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/myersguo/cc-mono/pkg/ai"
)

func TestNewProvider(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		provider, err := NewProvider(Config{
			APIKey: "test-key",
		})

		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.Name() != "openai" {
			t.Errorf("Expected provider name 'openai', got '%s'", provider.Name())
		}

		if provider.config.BaseURL != DefaultBaseURL {
			t.Errorf("Expected default base URL, got '%s'", provider.config.BaseURL)
		}
	})

	t.Run("CustomBaseURL", func(t *testing.T) {
		customURL := "https://custom.api.com/v1"
		provider, err := NewProvider(Config{
			APIKey:  "test-key",
			BaseURL: customURL,
		})

		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.config.BaseURL != customURL {
			t.Errorf("Expected base URL '%s', got '%s'", customURL, provider.config.BaseURL)
		}
	})

	t.Run("MissingAPIKey", func(t *testing.T) {
		_, err := NewProvider(Config{})

		if err == nil {
			t.Error("Expected error for missing API key")
		}
	})
}

func TestProvider_ValidateModel(t *testing.T) {
	provider, _ := NewProvider(Config{APIKey: "test-key"})

	t.Run("ValidModel", func(t *testing.T) {
		model := ai.Model{
			ID:       "gpt-4",
			Provider: "openai",
		}

		if err := provider.ValidateModel(model); err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("InvalidProvider", func(t *testing.T) {
		model := ai.Model{
			ID:       "claude-3",
			Provider: "anthropic",
		}

		if err := provider.ValidateModel(model); err == nil {
			t.Error("Expected error for invalid provider")
		}
	})
}

func TestProvider_Stream(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		// Parse request body
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Model != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%s'", req.Model)
		}

		if !req.Stream {
			t.Error("Expected stream to be true")
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send chunks
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" World"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	// Create provider with mock server URL
	provider, err := NewProvider(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Create context
	ctx := context.Background()
	model := ai.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	aiContext := ai.NewContext("", []ai.Message{
		ai.NewUserTextMessage("Hello"),
	})

	// Call stream
	stream := provider.Stream(ctx, model, aiContext, nil)

	// Collect events
	var events []ai.AssistantMessageEvent
	for event := range stream.Events() {
		events = append(events, event)
	}

	// Get result
	result := <-stream.Result()

	// Verify events
	if len(events) == 0 {
		t.Error("Expected events, got none")
	}

	// Verify result
	if result.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", result.Provider)
	}

	if len(result.Content) == 0 {
		t.Error("Expected content in result")
	}

	// Check text content
	if textContent, ok := result.Content[0].(ai.TextContent); ok {
		if !strings.Contains(textContent.Text, "Hello") {
			t.Errorf("Expected 'Hello' in content, got: %s", textContent.Text)
		}
	} else {
		t.Error("Expected text content")
	}

	// Verify usage
	if result.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", result.Usage.TotalTokens)
	}
}

func TestProvider_StreamWithToolCalls(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send tool call chunks
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"id":"call_123","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	// Create provider
	provider, err := NewProvider(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gpt-4",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Create context with tools
	ctx := context.Background()
	model := ai.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	aiContext := ai.NewContext("", []ai.Message{
		ai.NewUserTextMessage("Read test.txt"),
	})

	tools := []ai.Tool{
		ai.NewTool("read_file", "Read a file", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		}),
	}

	options := &ai.StreamOptions{
		Tools: tools,
	}

	// Call stream
	stream := provider.Stream(ctx, model, aiContext, options)

	// Collect events
	var hasToolCall bool
	for event := range stream.Events() {
		if event.Type == ai.EventTypeToolCall {
			hasToolCall = true
			if event.ToolCall.Name != "read_file" {
				t.Errorf("Expected tool call 'read_file', got '%s'", event.ToolCall.Name)
			}
		}
	}

	if !hasToolCall {
		t.Error("Expected tool call event")
	}

	// Get result
	result := <-stream.Result()

	// Verify stop reason
	if result.StopReason != ai.StopReasonToolUse {
		t.Errorf("Expected stop reason %s, got %s", ai.StopReasonToolUse, result.StopReason)
	}
}

func TestProvider_StreamError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"Invalid request","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	// Create provider
	provider, err := NewProvider(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Create context
	ctx := context.Background()
	model := ai.Model{
		ID:       "gpt-4",
		Provider: "openai",
	}
	aiContext := ai.NewContext("", []ai.Message{
		ai.NewUserTextMessage("Hello"),
	})

	// Call stream
	stream := provider.Stream(ctx, model, aiContext, nil)

	// Drain events
	for range stream.Events() {
	}

	// Check for error
	if stream.Error() == nil {
		t.Error("Expected error from stream")
	}
}

func TestProvider_Configuration(t *testing.T) {
	provider, _ := NewProvider(Config{
		APIKey:  "test-key",
		BaseURL: "https://test.com",
		Model:   "gpt-4",
	})

	t.Run("GetConfig", func(t *testing.T) {
		config := provider.GetConfig()
		if config.APIKey != "test-key" {
			t.Errorf("Expected API key 'test-key', got '%s'", config.APIKey)
		}
	})

	t.Run("SetAPIKey", func(t *testing.T) {
		provider.SetAPIKey("new-key")
		if provider.config.APIKey != "new-key" {
			t.Errorf("Expected API key 'new-key', got '%s'", provider.config.APIKey)
		}
	})

	t.Run("SetBaseURL", func(t *testing.T) {
		provider.SetBaseURL("https://new-url.com")
		if provider.config.BaseURL != "https://new-url.com" {
			t.Errorf("Expected base URL 'https://new-url.com', got '%s'", provider.config.BaseURL)
		}
	})

	t.Run("SetModel", func(t *testing.T) {
		provider.SetModel("gpt-3.5-turbo")
		if provider.config.Model != "gpt-3.5-turbo" {
			t.Errorf("Expected model 'gpt-3.5-turbo', got '%s'", provider.config.Model)
		}
	})
}
