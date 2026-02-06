package compaction

import (
	"context"
	"fmt"
	"strings"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// Compactor handles context compaction
type Compactor struct {
	provider      ai.Provider
	model         ai.Model
	contextWindow int
	safetyMargin  int // Reserve tokens for response
}

// Config represents compaction configuration
type Config struct {
	ContextWindow   int     // Total context window size
	SafetyMargin    int     // Reserve tokens for response (default: 4096)
	CompactionRatio float64 // Trigger compaction at this ratio (default: 0.8)
}

// NewCompactor creates a new compactor
func NewCompactor(provider ai.Provider, model ai.Model, config Config) *Compactor {
	if config.SafetyMargin == 0 {
		config.SafetyMargin = 4096
	}

	return &Compactor{
		provider:      provider,
		model:         model,
		contextWindow: config.ContextWindow,
		safetyMargin:  config.SafetyMargin,
	}
}

// NeedsCompaction checks if compaction is needed
func (c *Compactor) NeedsCompaction(messages []agent.AgentMessage, compactionRatio float64) bool {
	// Estimate token count (rough approximation: 1 token ~= 4 chars)
	totalChars := 0
	for _, msg := range messages {
		totalChars += estimateMessageSize(msg)
	}

	estimatedTokens := totalChars / 4
	threshold := int(float64(c.contextWindow-c.safetyMargin) * compactionRatio)

	return estimatedTokens > threshold
}

// Compact compacts the message history
func (c *Compactor) Compact(ctx context.Context, messages []agent.AgentMessage) ([]agent.AgentMessage, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Strategy: Keep first message (system prompt) and last N messages,
	// summarize the middle section

	// Find split points
	systemMessages := []agent.AgentMessage{}
	middleMessages := []agent.AgentMessage{}
	recentMessages := []agent.AgentMessage{}

	// Keep first system/user message
	if len(messages) > 0 {
		systemMessages = append(systemMessages, messages[0])
	}

	// Keep last 10 messages (approximately)
	keepRecentCount := 10
	if len(messages) > keepRecentCount+1 {
		middleMessages = messages[1 : len(messages)-keepRecentCount]
		recentMessages = messages[len(messages)-keepRecentCount:]
	} else {
		recentMessages = messages[1:]
	}

	// If no middle section to compact, return original
	if len(middleMessages) == 0 {
		return messages, nil
	}

	// Summarize middle section
	summary, err := c.summarizeMessages(ctx, middleMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize messages: %w", err)
	}

	// Construct compacted history
	compacted := []agent.AgentMessage{}
	compacted = append(compacted, systemMessages...)
	compacted = append(compacted, summary)
	compacted = append(compacted, recentMessages...)

	return compacted, nil
}

// summarizeMessages summarizes a section of messages
func (c *Compactor) summarizeMessages(ctx context.Context, messages []agent.AgentMessage) (agent.AgentMessage, error) {
	// Build context for summarization
	var transcript strings.Builder
	transcript.WriteString("Please provide a concise summary of the following conversation:\n\n")

	for _, msg := range messages {
		switch m := msg.Message.(type) {
		case ai.UserMessage:
			transcript.WriteString("User: ")
			transcript.WriteString(extractTextContent(m.Content))
			transcript.WriteString("\n\n")

		case ai.AssistantMessage:
			transcript.WriteString("Assistant: ")
			transcript.WriteString(extractTextContent(m.Content))

			// Include tool calls
			for _, content := range m.Content {
				if tc, ok := content.(ai.ToolCall); ok {
					transcript.WriteString(fmt.Sprintf("\n  [Tool: %s]", tc.Name))
				}
			}
			transcript.WriteString("\n\n")

		case ai.ToolResultMessage:
			transcript.WriteString(fmt.Sprintf("[Tool Result: %s]\n", m.ToolName))
			transcript.WriteString(extractTextContent(m.Content))
			transcript.WriteString("\n\n")
		}
	}

	// Request summary from LLM
	userMsg := ai.UserMessage{
		Type:    ai.MessageTypeUser,
		Content: []ai.Content{ai.NewTextContent(transcript.String())},
	}

	summaryContext := ai.Context{
		SystemPrompt: "You are a helpful assistant that summarizes conversations concisely. Focus on key decisions, actions taken, and important outcomes.",
		Messages:     []ai.Message{userMsg},
	}

	// Use simple stream for summarization
	stream := c.provider.StreamSimple(ctx, c.model, summaryContext, nil)

	// Wait for result
	result := <-stream.Result()

	// Check for errors
	if err := stream.Error(); err != nil {
		return agent.AgentMessage{}, err
	}

	// Create summary message
	summaryText := extractTextContent(result.Content)
	summaryPrefix := fmt.Sprintf("[Summary of %d messages]:\n", len(messages))

	summaryMessage := agent.AgentMessage{
		Message: ai.UserMessage{
			Type: ai.MessageTypeUser,
			Content: []ai.Content{
				ai.NewTextContent(summaryPrefix + summaryText),
			},
		},
		ID:        "compaction-summary",
		CreatedAt: messages[0].CreatedAt, // Use timestamp of first message
	}

	return summaryMessage, nil
}

// extractTextContent extracts text from content array
func extractTextContent(contents []ai.Content) string {
	var text strings.Builder
	for _, content := range contents {
		if tc, ok := content.(ai.TextContent); ok {
			text.WriteString(tc.Text)
		}
	}
	return text.String()
}

// estimateMessageSize estimates the size of a message in characters
func estimateMessageSize(msg agent.AgentMessage) int {
	totalChars := 0

	switch m := msg.Message.(type) {
	case ai.UserMessage:
		totalChars += len(extractTextContent(m.Content))

	case ai.AssistantMessage:
		totalChars += len(extractTextContent(m.Content))
		// Add overhead for tool calls
		for _, content := range m.Content {
			if tc, ok := content.(ai.ToolCall); ok {
				totalChars += len(tc.Name) + 50 // Rough estimate for tool call overhead
				for k, v := range tc.Params {
					totalChars += len(k) + len(fmt.Sprintf("%v", v))
				}
			}
		}

	case ai.ToolResultMessage:
		totalChars += len(m.ToolName) + len(extractTextContent(m.Content))
	}

	return totalChars
}
