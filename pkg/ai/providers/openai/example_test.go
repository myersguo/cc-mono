package openai_test

import (
	"context"
	"fmt"
	"os"

	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/myersguo/cc-mono/pkg/ai/providers/openai"
)

// Example_basicUsage demonstrates basic usage of the OpenAI provider
func Example_basicUsage() {
	// Create provider with OpenAI
	provider, err := openai.NewProvider(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	// Create model
	model := ai.Model{
		ID:       "gpt-4-turbo",
		Provider: "openai",
	}

	// Create context
	ctx := context.Background()
	aiContext := ai.NewContext("You are a helpful assistant", []ai.Message{
		ai.NewUserTextMessage("Say hello!"),
	})

	// Stream response
	stream := provider.Stream(ctx, model, aiContext, nil)

	// Process events
	for event := range stream.Events() {
		if event.Type == ai.EventTypeContentDelta {
			fmt.Print(event.TextDelta)
		}
	}

	// Get final result
	result := <-stream.Result()
	fmt.Printf("\nTokens used: %d\n", result.Usage.TotalTokens)
}

// Example_customBaseURL demonstrates using a custom base URL (e.g., for DeepSeek)
func Example_customBaseURL() {
	// Use DeepSeek API (OpenAI-compatible)
	provider, err := openai.NewProvider(openai.Config{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
	})
	if err != nil {
		panic(err)
	}

	// Create model
	model := ai.Model{
		ID:       "deepseek-chat",
		Provider: "openai", // Still use "openai" provider type
	}

	// Use the provider
	ctx := context.Background()
	aiContext := ai.NewContext("", []ai.Message{
		ai.NewUserTextMessage("Hello DeepSeek!"),
	})

	stream := provider.StreamSimple(ctx, model, aiContext, nil)

	for event := range stream.Events() {
		if event.Type == ai.EventTypeContentDelta {
			fmt.Print(event.TextDelta)
		}
	}

	<-stream.Result()
}

// Example_withTools demonstrates using tools with the OpenAI provider
func Example_withTools() {
	provider, err := openai.NewProvider(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		panic(err)
	}

	// Define tools
	tools := []ai.Tool{
		ai.NewTool("get_weather", "Get the weather for a location", map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "The city and state, e.g. San Francisco, CA",
				},
			},
			"required": []string{"location"},
		}),
	}

	// Create context
	ctx := context.Background()
	model := ai.Model{ID: "gpt-4-turbo", Provider: "openai"}
	aiContext := ai.NewContext("", []ai.Message{
		ai.NewUserTextMessage("What's the weather in San Francisco?"),
	})

	// Stream with tools
	stream := provider.Stream(ctx, model, aiContext, &ai.StreamOptions{
		Tools: tools,
	})

	// Process events
	for event := range stream.Events() {
		switch event.Type {
		case ai.EventTypeContentDelta:
			fmt.Print(event.TextDelta)
		case ai.EventTypeToolCall:
			fmt.Printf("\nTool call: %s(%v)\n", event.ToolCall.Name, event.ToolCall.Params)
		}
	}

	result := <-stream.Result()
	fmt.Printf("Stop reason: %s\n", result.StopReason)
}

// Example_multipleProviders demonstrates using multiple OpenAI-compatible providers
func Example_multipleProviders() {
	// OpenAI
	openaiProvider, _ := openai.NewProvider(openai.Config{
		APIKey: os.Getenv("OPENAI_API_KEY"),
		Model:  "gpt-4-turbo",
	})

	// DeepSeek
	deepseekProvider, _ := openai.NewProvider(openai.Config{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: "https://api.deepseek.com/v1",
		Model:   "deepseek-chat",
	})

	// Qwen (Alibaba Cloud)
	qwenProvider, _ := openai.NewProvider(openai.Config{
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Model:   "qwen-turbo",
	})

	// Register all providers
	ai.RegisterProvider(openaiProvider)
	ai.RegisterProvider(deepseekProvider)
	ai.RegisterProvider(qwenProvider)

	// List all providers
	providers := ai.ListProviders()
	fmt.Printf("Available providers: %v\n", providers)
}
