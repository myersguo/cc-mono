package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// MessageView renders a message
type MessageView struct {
	styles   *Styles
	expanded map[string]bool // Tool call IDs that are expanded
}

// NewMessageView creates a new message view
func NewMessageView(styles *Styles) *MessageView {
	return &MessageView{
		styles:   styles,
		expanded: make(map[string]bool),
	}
}

// Render renders an agent message
func (mv *MessageView) Render(msg agent.AgentMessage, width int) string {
	message := msg.Message

	switch message.GetType() {
	case ai.MessageTypeUser:
		return mv.renderUserMessage(message.(ai.UserMessage), width)
	case ai.MessageTypeAssistant:
		return mv.renderAssistantMessage(message.(ai.AssistantMessage), width)
	case ai.MessageTypeToolResult:
		return mv.renderToolResultMessage(message.(ai.ToolResultMessage), width)
	default:
		return ""
	}
}

// renderUserMessage renders a user message
func (mv *MessageView) renderUserMessage(msg ai.UserMessage, width int) string {
	var parts []string

	// Role header
	roleHeader := mv.styles.MessageRole.Render("You")
	timestamp := mv.styles.MessageTimestamp.Render(
		time.UnixMilli(msg.Timestamp).Format("15:04:05"),
	)
	header := lipgloss.JoinHorizontal(lipgloss.Top, roleHeader, " ", timestamp)
	parts = append(parts, header)

	// Content
	content := mv.renderContent(msg.Content, width-4)
	parts = append(parts, content)

	// Combine and apply style
	combined := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return mv.styles.UserMessage.Width(width - 4).Render(combined)
}

// renderAssistantMessage renders an assistant message
func (mv *MessageView) renderAssistantMessage(msg ai.AssistantMessage, width int) string {
	var parts []string

	// Only show role header if model name is present
	// (Empty model name is used for tool call display messages)
	if msg.Model != "" {
		modelName := msg.Model
		roleHeader := mv.styles.MessageRole.Render(modelName)
		timestamp := mv.styles.MessageTimestamp.Render(
			time.UnixMilli(msg.Timestamp).Format("15:04:05"),
		)
		header := lipgloss.JoinHorizontal(lipgloss.Top, roleHeader, " ", timestamp)
		parts = append(parts, header)
	}

	// Content
	content := mv.renderContent(msg.Content, width-4)
	parts = append(parts, content)

	// Usage info (if available)
	if msg.Usage.InputTokens > 0 || msg.Usage.OutputTokens > 0 {
		usage := mv.styles.HelpKey.Render(fmt.Sprintf(
			"Tokens: %d in, %d out",
			msg.Usage.InputTokens,
			msg.Usage.OutputTokens,
		))
		parts = append(parts, usage)
	}

	// Combine and apply style
	combined := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return mv.styles.AIMessage.Width(width - 4).Render(combined)
}

// renderToolResultMessage renders a tool result message
func (mv *MessageView) renderToolResultMessage(msg ai.ToolResultMessage, width int) string {
	// Don't render tool results separately - they're shown inline in the response
	// Just return empty string or a minimal indicator
	return ""
}

// renderContent renders message content
func (mv *MessageView) renderContent(contents []ai.Content, width int) string {
	var parts []string

	for _, content := range contents {
		switch c := content.(type) {
		case ai.TextContent:
			parts = append(parts, mv.renderText(c.Text, width))
		case ai.ThinkingContent:
			parts = append(parts, mv.renderThinking(c.Thinking, width))
		case ai.ToolCall:
			parts = append(parts, mv.renderToolCall(c, width))
		case ai.ImageContent:
			parts = append(parts, mv.renderImage(c, width))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderText renders text content
func (mv *MessageView) renderText(text string, width int) string {
	// Simple markdown-like rendering
	var lines []string

	for _, line := range strings.Split(text, "\n") {
		// Code blocks (lines starting with 4 spaces or tab)
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			lines = append(lines, mv.styles.CodeBlock.Render(strings.TrimPrefix(strings.TrimPrefix(line, "    "), "\t")))
			continue
		}

		// Bullet points (lines starting with - or *)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			bullet := lipgloss.NewStyle().
				Foreground(mv.styles.Theme.Accent).
				Render("●")
			content := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			lines = append(lines, bullet+" "+content)
			continue
		}

		// Numbered lists (lines starting with number.)
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' {
			// Check if there's a ". " pattern in the first few characters
			checkLen := len(line)
			if checkLen > 4 {
				checkLen = 4
			}
			if strings.Contains(line[:checkLen], ". ") {
				// Keep numbered format
				lines = append(lines, line)
				continue
			}
		}

		// Inline code (backticks)
		if strings.Contains(line, "`") {
			line = mv.renderInlineCode(line)
		}

		// Quotes (lines starting with >)
		if strings.HasPrefix(line, ">") {
			lines = append(lines, mv.styles.Quote.Render(strings.TrimPrefix(line, "> ")))
			continue
		}

		// Bold text (**text**)
		if strings.Contains(line, "**") {
			line = mv.renderBoldText(line)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderInlineCode renders inline code with backticks
func (mv *MessageView) renderInlineCode(text string) string {
	var result strings.Builder
	inCode := false
	var currentCode strings.Builder

	for i := 0; i < len(text); i++ {
		if text[i] == '`' {
			if inCode {
				// End of code block
				result.WriteString(mv.styles.InlineCode.Render(currentCode.String()))
				currentCode.Reset()
				inCode = false
			} else {
				// Start of code block
				inCode = true
			}
		} else {
			if inCode {
				currentCode.WriteByte(text[i])
			} else {
				result.WriteByte(text[i])
			}
		}
	}

	// Handle unclosed code block
	if inCode {
		result.WriteString(currentCode.String())
	}

	return result.String()
}

// renderBoldText renders bold text with **text**
func (mv *MessageView) renderBoldText(text string) string {
	var result strings.Builder
	inBold := false
	var currentBold strings.Builder
	i := 0

	for i < len(text) {
		if i+1 < len(text) && text[i:i+2] == "**" {
			if inBold {
				// End of bold
				boldStyle := lipgloss.NewStyle().Bold(true)
				result.WriteString(boldStyle.Render(currentBold.String()))
				currentBold.Reset()
				inBold = false
			} else {
				// Start of bold
				inBold = true
			}
			i += 2
		} else {
			if inBold {
				currentBold.WriteByte(text[i])
			} else {
				result.WriteByte(text[i])
			}
			i++
		}
	}

	// Handle unclosed bold
	if inBold {
		result.WriteString("**")
		result.WriteString(currentBold.String())
	}

	return result.String()
}

// renderThinking renders thinking content
func (mv *MessageView) renderThinking(thinking string, width int) string {
	// Simple format: * Thinking...
	prefix := lipgloss.NewStyle().
		Foreground(mv.styles.Theme.Accent).
		Render("* Thinking...")

	// Split thinking into lines and check for title
	lines := strings.Split(thinking, "\n")
	var formattedLines []string
	formattedLines = append(formattedLines, prefix)
	formattedLines = append(formattedLines, "")

	// Render thinking content with slight dimming
	for _, line := range lines {
		if line != "" {
			styled := lipgloss.NewStyle().
				Foreground(mv.styles.Theme.Muted).
				Render(line)
			formattedLines = append(formattedLines, styled)
		} else {
			formattedLines = append(formattedLines, "")
		}
	}

	return strings.Join(formattedLines, "\n")
}

// renderToolCall renders a tool call
func (mv *MessageView) renderToolCall(toolCall ai.ToolCall, width int) string {
	// Simple format: ● toolname(param1: value1, param2: value2)
	bullet := lipgloss.NewStyle().
		Foreground(mv.styles.Theme.Secondary).
		Render("●")

	toolName := lipgloss.NewStyle().
		Bold(true).
		Foreground(mv.styles.Theme.Secondary).
		Render(toolCall.Name)

	// Format parameters inline
	var paramStrs []string
	for key, value := range toolCall.Params {
		valueStr := fmt.Sprintf("%v", value)
		// Truncate very long values
		if len(valueStr) > 100 {
			valueStr = valueStr[:100] + "..."
		}

		keyStyled := lipgloss.NewStyle().
			Foreground(mv.styles.Theme.Accent).
			Render(key)

		paramStrs = append(paramStrs, fmt.Sprintf("%s: %s", keyStyled, valueStr))
	}

	params := ""
	if len(paramStrs) > 0 {
		params = "(" + strings.Join(paramStrs, ", ") + ")"
	} else {
		params = "()"
	}

	return fmt.Sprintf("%s %s%s", bullet, toolName, params)
}

// renderImage renders an image placeholder
func (mv *MessageView) renderImage(img ai.ImageContent, width int) string {
	// For now, just show a placeholder
	// TODO: Implement Kitty/iTerm2 protocol for actual image display
	return mv.styles.HelpKey.Render(fmt.Sprintf("📷 Image (%s)", img.Source.MediaType))
}

// ToggleExpanded toggles the expanded state of a tool result
func (mv *MessageView) ToggleExpanded(toolCallID string) {
	mv.expanded[toolCallID] = !mv.expanded[toolCallID]
}

// IsExpanded checks if a tool result is expanded
func (mv *MessageView) IsExpanded(toolCallID string) bool {
	return mv.expanded[toolCallID]
}

// RenderToolCallDisplay formats a tool call for display (used in ToolExecutionStartEvent)
func (mv *MessageView) RenderToolCallDisplay(toolName string, args map[string]interface{}) string {
	// Format: > toolname(param: value, ...)
	prefix := lipgloss.NewStyle().
		Foreground(mv.styles.Theme.Secondary).
		Render(">")

	toolNameStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(mv.styles.Theme.Secondary).
		Render(toolName)

	// Format parameters
	var paramStrs []string
	for key, value := range args {
		valueStr := fmt.Sprintf("%v", value)
		// Truncate very long values
		if len(valueStr) > 100 {
			valueStr = valueStr[:100] + "..."
		}

		keyStyled := lipgloss.NewStyle().
			Foreground(mv.styles.Theme.Accent).
			Render(key)

		paramStrs = append(paramStrs, fmt.Sprintf("%s: %s", keyStyled, valueStr))
	}

	params := ""
	if len(paramStrs) > 0 {
		params = "(" + strings.Join(paramStrs, ", ") + ")"
	} else {
		params = "()"
	}

	return fmt.Sprintf("%s %s%s", prefix, toolNameStyled, params)
}
