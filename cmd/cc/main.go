package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/myersguo/cc-mono/internal/tui"
	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/codingagent/extensions"
)

func main() {
	// Create context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Execute CLI
	if err := Execute(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runTUI starts the bubbletea TUI interface
func runTUI(
	agentInst *agent.Agent,
	theme string,
	extensionRunner *extensions.Runner,
) error {
	// Create chat model
	chatModel := tui.NewChatModel(agentInst, theme)

	// Create bubbletea program
	p := tea.NewProgram(
		chatModel,
		// tea.WithAltScreen(), // Disable alt screen to allow terminal copy/paste
		// tea.WithMouseCellMotion(), // Disable mouse to avoid interference with selection
	)

	// Run TUI
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Call extension OnAgentEnd when TUI exits
	if extensionRunner != nil {
		if err := extensionRunner.OnAgentEnd(context.Background()); err != nil {
			// Log error but don't fail
			fmt.Fprintf(os.Stderr, "Warning: extension cleanup error: %v\n", err)
		}
	}

	return nil
}
