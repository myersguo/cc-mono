package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/myersguo/cc-mono/pkg/ai"
)

const (
	// DefaultBaseURL is the default OpenAI API base URL
	DefaultBaseURL = "https://api.openai.com/v1"

	// DefaultModel is the default model to use
	DefaultModel = "gpt-4-turbo"
)

// Config represents OpenAI provider configuration
type Config struct {
	APIKey  string // API key for authentication
	BaseURL string // Base URL for API (default: https://api.openai.com/v1)
	Model   string // Default model name
}

// Provider implements the OpenAI provider
type Provider struct {
	*ai.BaseProvider
	config     Config
	httpClient *http.Client
}

// NewProvider creates a new OpenAI provider
func NewProvider(config Config) (*Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}

	if config.Model == "" {
		config.Model = DefaultModel
	}

	// Create default model
	defaultModel := ai.Model{
		ID:              config.Model,
		Provider:        "openai",
		Name:            config.Model,
		ContextWindow:   128000,
		MaxOutput:       4096,
		InputCostPer1M:  10.0,
		OutputCostPer1M: 30.0,
		SupportsVision:  true,
		SupportsTools:   true,
	}

	return &Provider{
		BaseProvider: ai.NewBaseProvider("openai", defaultModel),
		config:       config,
		httpClient:   &http.Client{},
	}, nil
}

// Stream sends a request and returns a stream of events
func (p *Provider) Stream(
	ctx context.Context,
	model ai.Model,
	context ai.Context,
	options *ai.StreamOptions,
) *ai.AssistantMessageEventStream {
	stream := ai.NewAssistantMessageEventStream(ctx)

	go func() {
		defer stream.Close()

		// Convert to OpenAI request
		req, err := convertContextToRequest(model, context, options)
		if err != nil {
			stream.SendError(fmt.Errorf("failed to convert context: %w", err))
			return
		}

		// Make API call
		if err := p.streamRequest(ctx, req, stream); err != nil {
			stream.SendError(err)
			return
		}
	}()

	return stream
}

// StreamSimple sends a simple request without tools
func (p *Provider) StreamSimple(
	ctx context.Context,
	model ai.Model,
	context ai.Context,
	options *ai.SimpleStreamOptions,
) *ai.AssistantMessageEventStream {
	// Convert to full options
	fullOptions := &ai.StreamOptions{
		Temperature: options.Temperature,
		MaxTokens:   options.MaxTokens,
	}

	return p.Stream(ctx, model, context, fullOptions)
}

// ValidateModel checks if the model is supported
func (p *Provider) ValidateModel(model ai.Model) error {
	if model.Provider != "openai" {
		return fmt.Errorf("model provider must be 'openai', got '%s'", model.Provider)
	}
	return nil
}

// streamRequest makes the streaming API request
func (p *Provider) streamRequest(
	ctx context.Context,
	req *ChatCompletionRequest,
	stream *ai.AssistantMessageEventStream,
) error {
	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(p.config.BaseURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	httpReq.Header.Set("Accept", "text/event-stream")

	// Make request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		return fmt.Errorf("API error: status %d: %s", resp.StatusCode, string(body))
	}

	// Send start event
	stream.SendEvent(ai.NewStartEvent())

	// Process SSE stream
	if err := p.processSSEStream(resp.Body, stream); err != nil {
		return fmt.Errorf("failed to process stream: %w", err)
	}

	return nil
}

// processSSEStream processes the Server-Sent Events stream
func (p *Provider) processSSEStream(
	reader io.Reader,
	stream *ai.AssistantMessageEventStream,
) error {
	scanner := bufio.NewScanner(reader)

	// Accumulate content and tool calls
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	// Map to accumulate tool call arguments by index
	toolCallsMap := make(map[int]*ToolCall)
	var usage ai.Usage
	var stopReason ai.StopReason = ai.StopReasonEndTurn

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE format: "data: <json>"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for [DONE] signal
		if data == "[DONE]" {
			break
		}

		// Parse chunk
		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Log error but continue processing
			continue
		}

		// Accumulate tool calls and reasoning from chunk delta
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta

			// Accumulate reasoning content (for o1 models)
			if delta.ReasoningContent != "" {
				reasoningBuilder.WriteString(delta.ReasoningContent)
			}

			for _, tc := range delta.ToolCalls {
				// Use a synthetic index if not provided
				// In OpenAI streaming, tool calls come with their full data eventually
				idx := 0 // Simplified: assume single tool call for now

				if existing, ok := toolCallsMap[idx]; ok {
					// Accumulate arguments
					existing.Function.Arguments += tc.Function.Arguments
				} else {
					// Initialize new tool call
					toolCallsMap[idx] = &ToolCall{
						ID:   tc.ID,
						Type: tc.Type,
						Function: FunctionCall{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}
		}

		// Convert chunk to events
		events, err := convertChunkToEvent(chunk)
		if err != nil {
			return fmt.Errorf("failed to convert chunk: %w", err)
		}

		// Process events
		for _, event := range events {
			// Accumulate content for final message
			switch event.Type {
			case ai.EventTypeContentDelta:
				if event.ContentType == ai.ContentTypeText {
					contentBuilder.WriteString(event.TextDelta)
				} else if event.ContentType == ai.ContentTypeThinking {
					reasoningBuilder.WriteString(event.ThinkingDelta)
				}
			case ai.EventTypeUsage:
				if event.Usage != nil {
					usage = *event.Usage
				}
			case ai.EventTypeEnd:
				stopReason = event.StopReason
			}

			// Send event
			if err := stream.SendEvent(event); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Build final message
	finalContent := make([]ai.Content, 0)

	// Add reasoning/thinking content first (if present)
	if reasoningBuilder.Len() > 0 {
		finalContent = append(finalContent, ai.NewThinkingContent(reasoningBuilder.String()))
	}

	if contentBuilder.Len() > 0 {
		finalContent = append(finalContent, ai.NewTextContent(contentBuilder.String()))
	}

	// Parse accumulated tool calls
	for _, tc := range toolCallsMap {
		// Parse the accumulated arguments
		var params map[string]any
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				// Log error but continue
				fmt.Fprintf(os.Stderr, "[WARN] Failed to parse tool arguments: %v\n", err)
				continue
			}
		}

		toolCall := ai.NewToolCall(tc.ID, tc.Function.Name, params)
		finalContent = append(finalContent, toolCall)
	}

	result := ai.NewAssistantMessage(
		finalContent,
		"openai",
		"openai",
		p.config.Model,
		usage,
		stopReason,
	)

	// Send final result
	return stream.SendResult(result)
}

// SetAPIKey updates the API key
func (p *Provider) SetAPIKey(apiKey string) {
	p.config.APIKey = apiKey
}

// SetBaseURL updates the base URL
func (p *Provider) SetBaseURL(baseURL string) {
	p.config.BaseURL = baseURL
}

// SetModel updates the default model
func (p *Provider) SetModel(model string) {
	p.config.Model = model
}

// GetConfig returns the current configuration
func (p *Provider) GetConfig() Config {
	return p.config
}
