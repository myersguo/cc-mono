package openai

import (
	"encoding/json"
	"fmt"

	"github.com/myersguo/cc-mono/pkg/ai"
)

// convertContextToRequest converts our Context to OpenAI ChatCompletionRequest
func convertContextToRequest(
	model ai.Model,
	context ai.Context,
	options *ai.StreamOptions,
) (*ChatCompletionRequest, error) {
	// Convert messages
	messages := make([]ChatMessage, 0)

	// Add system message if present
	if context.SystemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: context.SystemPrompt,
		})
	}

	// Convert context messages
	for _, msg := range context.Messages {
		converted, err := convertMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		messages = append(messages, converted...)
	}

	// Build request
	req := &ChatCompletionRequest{
		Model:    model.ID,
		Messages: messages,
		Stream:   true,
		StreamOptions: &StreamOptions{
			IncludeUsage: true,
		},
	}

	// Add options
	if options != nil {
		if options.Temperature != nil {
			req.Temperature = options.Temperature
		}
		if options.MaxTokens != nil {
			req.MaxTokens = options.MaxTokens
		}

		// Convert tools
		if len(options.Tools) > 0 {
			req.Tools = make([]ChatTool, len(options.Tools))
			for i, tool := range options.Tools {
				req.Tools[i] = ChatTool{
					Type: "function",
					Function: ToolFunction{
						Name:        tool.Name,
						Description: tool.Description,
						Parameters:  tool.Parameters,
					},
				}
			}
		}
	}

	return req, nil
}

// convertMessage converts a single message to OpenAI format
func convertMessage(msg ai.Message) ([]ChatMessage, error) {
	switch m := msg.(type) {
	case ai.UserMessage:
		return convertUserMessage(m)
	case ai.AssistantMessage:
		return convertAssistantMessage(m)
	case ai.ToolResultMessage:
		return convertToolResultMessage(m)
	default:
		return nil, fmt.Errorf("unknown message type: %T", msg)
	}
}

// convertUserMessage converts UserMessage to OpenAI format
func convertUserMessage(msg ai.UserMessage) ([]ChatMessage, error) {
	// Check if we have only text content
	hasOnlyText := true
	for _, content := range msg.Content {
		if content.ContentType() != ai.ContentTypeText {
			hasOnlyText = false
			break
		}
	}

	// Simple text-only message
	if hasOnlyText && len(msg.Content) == 1 {
		text := msg.Content[0].(ai.TextContent)
		return []ChatMessage{{
			Role:    "user",
			Content: text.Text,
		}}, nil
	}

	// Multimodal content
	parts := make([]ContentPart, 0)
	for _, content := range msg.Content {
		switch c := content.(type) {
		case ai.TextContent:
			parts = append(parts, ContentPart{
				Type: "text",
				Text: c.Text,
			})
		case ai.ImageContent:
			url := c.Source.URL
			if c.Source.Type == "base64" {
				url = fmt.Sprintf("data:%s;base64,%s", c.Source.MediaType, c.Source.Data)
			}
			parts = append(parts, ContentPart{
				Type: "image_url",
				ImageURL: &ImageURL{
					URL:    url,
					Detail: "auto",
				},
			})
		}
	}

	return []ChatMessage{{
		Role:    "user",
		Content: parts,
	}}, nil
}

// convertAssistantMessage converts AssistantMessage to OpenAI format
func convertAssistantMessage(msg ai.AssistantMessage) ([]ChatMessage, error) {
	result := ChatMessage{
		Role: "assistant",
	}

	// Separate text content and tool calls
	var textParts []string
	var toolCalls []ToolCall

	for _, content := range msg.Content {
		switch c := content.(type) {
		case ai.TextContent:
			textParts = append(textParts, c.Text)
		case ai.ThinkingContent:
			// OpenAI doesn't have thinking content, skip or include as text
			// For now, skip it
		case ai.ToolCall:
			// Convert params to JSON string
			paramsJSON, err := json.Marshal(c.Params)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tool params: %w", err)
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      c.Name,
					Arguments: string(paramsJSON),
				},
			})
		}
	}

	// Set content
	if len(textParts) > 0 {
		result.Content = textParts[0] // OpenAI expects single string
		if len(textParts) > 1 {
			// Concatenate multiple text parts
			fullText := ""
			for _, part := range textParts {
				fullText += part
			}
			result.Content = fullText
		}
	}

	// Set tool calls
	if len(toolCalls) > 0 {
		result.ToolCalls = toolCalls
	}

	return []ChatMessage{result}, nil
}

// convertToolResultMessage converts ToolResultMessage to OpenAI format
func convertToolResultMessage(msg ai.ToolResultMessage) ([]ChatMessage, error) {
	// Extract text content
	var textContent string
	for _, content := range msg.Content {
		if text, ok := content.(ai.TextContent); ok {
			textContent += text.Text
		}
	}

	return []ChatMessage{{
		Role:       "tool",
		Content:    textContent,
		ToolCallID: msg.ToolCallID,
	}}, nil
}

// convertChunkToEvent converts OpenAI chunk to our event
func convertChunkToEvent(chunk ChatCompletionChunk) ([]ai.AssistantMessageEvent, error) {
	events := make([]ai.AssistantMessageEvent, 0)

	if len(chunk.Choices) == 0 {
		return events, nil
	}

	delta := chunk.Choices[0].Delta
	finishReason := chunk.Choices[0].FinishReason

	// Handle reasoning content (for o1 models)
	if delta.ReasoningContent != "" {
		events = append(events, ai.NewThinkingDeltaEvent(delta.ReasoningContent))
	}

	// Handle content delta
	if delta.Content != nil {
		if text, ok := delta.Content.(string); ok && text != "" {
			events = append(events, ai.NewTextDeltaEvent(text))
		}
	}

	// Handle tool calls
	// Note: In streaming, tool call arguments arrive in chunks and may not be valid JSON yet.
	// We skip tool call events during streaming and only emit them in the final message.
	// This avoids "unexpected end of JSON input" errors.
	if len(delta.ToolCalls) > 0 {
		// Skip tool call events in streaming chunks
		// Tool calls will be accumulated and parsed in the final message
	}

	// Handle usage
	if chunk.Usage != nil {
		events = append(events, ai.NewUsageEvent(ai.Usage{
			InputTokens:  chunk.Usage.PromptTokens,
			OutputTokens: chunk.Usage.CompletionTokens,
			TotalTokens:  chunk.Usage.TotalTokens,
		}))
	}

	// Handle finish reason
	if finishReason != nil && *finishReason != "" {
		stopReason := convertFinishReason(*finishReason)
		events = append(events, ai.NewEndEvent(stopReason))
	}

	return events, nil
}

// convertFinishReason converts OpenAI finish reason to our StopReason
func convertFinishReason(reason string) ai.StopReason {
	switch reason {
	case "stop":
		return ai.StopReasonEndTurn
	case "length":
		return ai.StopReasonMaxTokens
	case "tool_calls":
		return ai.StopReasonToolUse
	case "content_filter":
		return ai.StopReasonError
	default:
		return ai.StopReasonEndTurn
	}
}
