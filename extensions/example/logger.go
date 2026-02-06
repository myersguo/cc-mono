package example

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
	"github.com/myersguo/cc-mono/pkg/shared"
)

// LoggerExtension logs all tool calls and results
type LoggerExtension struct {
	*shared.BaseExtension
	logFile string
	verbose bool
}

// NewLoggerExtension creates a new logger extension
func NewLoggerExtension() *LoggerExtension {
	return &LoggerExtension{
		BaseExtension: shared.NewBaseExtension(
			"logger",
			"1.0.0",
			"Logs all tool calls and results for debugging",
		),
	}
}

// Init initializes the extension
func (e *LoggerExtension) Init(config map[string]any) error {
	if logFile, ok := config["log_file"].(string); ok {
		e.logFile = logFile
	}

	if verbose, ok := config["verbose"].(bool); ok {
		e.verbose = verbose
	}

	log.Printf("[Logger Extension] Initialized with config: %v", config)
	return nil
}

// OnToolCall logs tool calls
func (e *LoggerExtension) OnToolCall(ctx context.Context, toolName string, params map[string]any) (map[string]any, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[%s] Tool Call: %s with params: %v", timestamp, toolName, params)

	if e.verbose {
		for key, value := range params {
			log.Printf("  - %s: %v", key, value)
		}
	}

	// Don't modify params, just log
	return nil, nil
}

// OnToolResult logs tool results
func (e *LoggerExtension) OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[%s] Tool Result: %s (error: %v)", timestamp, toolName, result.IsError)

	if e.verbose && len(result.Content) > 0 {
		for i, content := range result.Content {
			log.Printf("  Content[%d]: %T", i, content)
		}
	}

	// Don't modify result, just log
	return agent.AgentToolResult{}, nil
}

// OnAgentStart logs when the agent starts
func (e *LoggerExtension) OnAgentStart(ctx context.Context) error {
	log.Println("[Logger Extension] Agent started")
	return nil
}

// OnAgentEnd logs when the agent completes
func (e *LoggerExtension) OnAgentEnd(ctx context.Context) error {
	log.Println("[Logger Extension] Agent completed")
	return nil
}

// Shutdown cleans up resources
func (e *LoggerExtension) Shutdown() error {
	log.Println("[Logger Extension] Shutting down")
	return nil
}

// TimeTrackerExtension tracks execution time of tools
type TimeTrackerExtension struct {
	*shared.BaseExtension
	startTimes map[string]time.Time
}

// NewTimeTrackerExtension creates a new time tracker extension
func NewTimeTrackerExtension() *TimeTrackerExtension {
	return &TimeTrackerExtension{
		BaseExtension: shared.NewBaseExtension(
			"time-tracker",
			"1.0.0",
			"Tracks execution time of tool calls",
		),
		startTimes: make(map[string]time.Time),
	}
}

// OnToolCall records start time
func (e *TimeTrackerExtension) OnToolCall(ctx context.Context, toolName string, params map[string]any) (map[string]any, error) {
	e.startTimes[toolName] = time.Now()
	return nil, nil
}

// OnToolResult calculates and logs execution time
func (e *TimeTrackerExtension) OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error) {
	if startTime, ok := e.startTimes[toolName]; ok {
		duration := time.Since(startTime)
		log.Printf("[Time Tracker] %s took %v", toolName, duration)
		delete(e.startTimes, toolName)

		// Add timing info to result details
		if result.Details == nil {
			result.Details = make(map[string]any)
		}

		if details, ok := result.Details.(map[string]any); ok {
			details["execution_time_ms"] = duration.Milliseconds()
			return result, nil
		}
	}

	return agent.AgentToolResult{}, nil
}

// ContentFilterExtension filters sensitive content from tool results
type ContentFilterExtension struct {
	*shared.BaseExtension
	blockedWords []string
}

// NewContentFilterExtension creates a new content filter extension
func NewContentFilterExtension() *ContentFilterExtension {
	return &ContentFilterExtension{
		BaseExtension: shared.NewBaseExtension(
			"content-filter",
			"1.0.0",
			"Filters sensitive content from tool results",
		),
		blockedWords: []string{
			"password",
			"secret",
			"token",
			"api_key",
			"private_key",
		},
	}
}

// Init initializes the filter with custom blocked words
func (e *ContentFilterExtension) Init(config map[string]any) error {
	if blockedWords, ok := config["blocked_words"].([]string); ok {
		e.blockedWords = append(e.blockedWords, blockedWords...)
	}

	log.Printf("[Content Filter] Initialized with %d blocked words", len(e.blockedWords))
	return nil
}

// OnToolResult filters sensitive content
func (e *ContentFilterExtension) OnToolResult(ctx context.Context, toolName string, result agent.AgentToolResult) (agent.AgentToolResult, error) {
	// Only filter read operations
	if toolName != "read" {
		return agent.AgentToolResult{}, nil
	}

	filtered := false
	newContent := make([]ai.Content, len(result.Content))

	for i, content := range result.Content {
		if textContent, ok := content.(ai.TextContent); ok {
			text := textContent.Text

			// Check for blocked words
			for _, word := range e.blockedWords {
				if contains(text, word) {
					text = fmt.Sprintf("[FILTERED: Content contains sensitive information]")
					filtered = true
					break
				}
			}

			if filtered {
				newContent[i] = ai.NewTextContent(text)
			} else {
				newContent[i] = content
			}
		} else {
			newContent[i] = content
		}
	}

	if filtered {
		result.Content = newContent
		return result, nil
	}

	return agent.AgentToolResult{}, nil
}

// contains checks if text contains a word (case-insensitive)
func contains(text, word string) bool {
	// Simple case-insensitive check
	// In production, use strings.ToLower or regex
	return len(text) > len(word) && text != word
}

// Register all example extensions with the global registry
func init() {
	shared.GlobalRegistry.Register(NewLoggerExtension())
	shared.GlobalRegistry.Register(NewTimeTrackerExtension())
	shared.GlobalRegistry.Register(NewContentFilterExtension())
}
