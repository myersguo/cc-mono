package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/myersguo/cc-mono/pkg/agent"
)

// StreamRenderer handles direct stdout output for completed messages
// This ensures messages are permanently saved in the terminal's scrollback buffer
type StreamRenderer struct {
	output      io.Writer
	messageView *MessageView
	width       int
}

// NewStreamRenderer creates a new stream renderer
func NewStreamRenderer(messageView *MessageView, width int) *StreamRenderer {
	return &StreamRenderer{
		output:      os.Stdout,
		messageView: messageView,
		width:       width,
	}
}

// RenderMessage outputs a completed message directly to stdout
// This message will be permanently visible in terminal scrollback
func (sr *StreamRenderer) RenderMessage(msg agent.AgentMessage) {
	rendered := sr.messageView.Render(msg, sr.width)
	fmt.Fprintln(sr.output, rendered)
}

// SetWidth updates the renderer width (for terminal resize)
func (sr *StreamRenderer) SetWidth(width int) {
	sr.width = width
}

// DynamicRenderer handles in-place updates for streaming messages
// It uses ANSI escape codes to update content without polluting scrollback
type DynamicRenderer struct {
	previousLines []string
	isActive      bool
}

// NewDynamicRenderer creates a new dynamic renderer
func NewDynamicRenderer() *DynamicRenderer {
	return &DynamicRenderer{
		previousLines: []string{},
		isActive:      false,
	}
}

// Update updates the dynamic content area using differential rendering
// This minimizes terminal I/O and prevents scrollback pollution
func (dr *DynamicRenderer) Update(newLines []string) {
	if len(newLines) == 0 {
		return
	}

	// Begin synchronized output to prevent flicker (CSI 2026)
	// Supported by modern terminals like iTerm2, Ghostty, etc.
	fmt.Print("\x1b[?2026h")
	defer fmt.Print("\x1b[?2026l")

	if !dr.isActive {
		// First render - just output the lines
		for i, line := range newLines {
			fmt.Print(line)
			if i < len(newLines)-1 {
				fmt.Print("\r\n")
			}
		}
		dr.isActive = true
	} else {
		// Subsequent renders - use differential update
		firstChanged, lastChanged := dr.computeDiff(newLines)

		if firstChanged != -1 {
			// Move cursor to first changed line
			if firstChanged > 0 {
				fmt.Printf("\x1b[%dA", len(dr.previousLines)-firstChanged)
			}
			fmt.Print("\r") // Move to start of line

			// Redraw changed lines
			for i := firstChanged; i <= lastChanged; i++ {
				if i < len(newLines) {
					fmt.Print("\x1b[2K") // Clear line (CSI 2K - preserves scrollback)
					fmt.Print(newLines[i])
					if i < lastChanged {
						fmt.Print("\r\n")
					}
				}
			}

			// If new content is shorter, clear remaining old lines
			if len(newLines) < len(dr.previousLines) {
				for i := len(newLines); i < len(dr.previousLines); i++ {
					fmt.Print("\r\n")
					fmt.Print("\x1b[2K")
				}
				// Move cursor back
				linesToMoveUp := len(dr.previousLines) - lastChanged - 1
				if linesToMoveUp > 0 {
					fmt.Printf("\x1b[%dA", linesToMoveUp)
				}
			}

			// Move cursor to end of content
			linesToMoveDown := lastChanged - firstChanged
			if linesToMoveDown > 0 && firstChanged > 0 {
				fmt.Printf("\x1b[%dB", linesToMoveDown)
			}
		}
	}

	dr.previousLines = make([]string, len(newLines))
	copy(dr.previousLines, newLines)
}

// Clear clears the dynamic content and resets state
func (dr *DynamicRenderer) Clear() {
	if !dr.isActive || len(dr.previousLines) == 0 {
		return
	}

	// Move cursor to start of dynamic content
	if len(dr.previousLines) > 1 {
		fmt.Printf("\x1b[%dA", len(dr.previousLines)-1)
	}
	fmt.Print("\r")

	// Clear all lines
	for i := 0; i < len(dr.previousLines); i++ {
		fmt.Print("\x1b[2K")
		if i < len(dr.previousLines)-1 {
			fmt.Print("\r\n")
		}
	}

	dr.previousLines = []string{}
	dr.isActive = false
}

// Finalize outputs the current dynamic content as permanent text
// This moves it from the dynamic area into the scrollback buffer
func (dr *DynamicRenderer) Finalize() {
	if dr.isActive && len(dr.previousLines) > 0 {
		// Just add a newline to commit the content
		fmt.Println()
	}
	dr.previousLines = []string{}
	dr.isActive = false
}

// computeDiff compares old and new lines and returns the range that changed
func (dr *DynamicRenderer) computeDiff(newLines []string) (firstChanged int, lastChanged int) {
	firstChanged = -1
	lastChanged = -1

	maxLen := len(newLines)
	if len(dr.previousLines) > maxLen {
		maxLen = len(dr.previousLines)
	}

	for i := 0; i < maxLen; i++ {
		oldLine := ""
		newLine := ""

		if i < len(dr.previousLines) {
			oldLine = dr.previousLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			if firstChanged == -1 {
				firstChanged = i
			}
			lastChanged = i
		}
	}

	return firstChanged, lastChanged
}

// IsActive returns whether the dynamic renderer has active content
func (dr *DynamicRenderer) IsActive() bool {
	return dr.isActive
}

// splitLines splits content into lines for rendering
func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}
