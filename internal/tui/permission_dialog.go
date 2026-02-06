package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/myersguo/cc-mono/pkg/agent"
)

// getUserName returns the current username
func getUserName() string {
	if username := os.Getenv("USER"); username != "" {
		return username
	}
	if username := os.Getenv("USERNAME"); username != "" {
		return username
	}
	return "user"
}

// PermissionDialogModel represents the permission confirmation dialog
type PermissionDialogModel struct {
	request       *agent.PermissionRequest
	styles        *Styles
	selectedIndex int
	width         int
	height        int
	visible       bool
}

// NewPermissionDialog creates a new permission dialog
func NewPermissionDialog(styles *Styles) *PermissionDialogModel {
	return &PermissionDialogModel{
		styles:        styles,
		selectedIndex: 0,
		visible:       false,
	}
}

// Show displays the permission dialog with a request
func (m *PermissionDialogModel) Show(req *agent.PermissionRequest) {
	m.request = req
	m.selectedIndex = 0
	m.visible = true
}

// Hide hides the permission dialog
func (m *PermissionDialogModel) Hide() {
	m.visible = false
	m.request = nil
}

// IsVisible returns whether the dialog is visible
func (m *PermissionDialogModel) IsVisible() bool {
	return m.visible
}

// GetRequest returns the current permission request
func (m *PermissionDialogModel) GetRequest() *agent.PermissionRequest {
	return m.request
}

// Update handles messages
func (m *PermissionDialogModel) Update(msg tea.Msg) (*PermissionDialogModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "down", "j":
			if m.selectedIndex < 2 {
				m.selectedIndex++
			}
		case "enter":
			// User made a choice
			return m, func() tea.Msg {
				return PermissionResponseMsg{
					RequestID: m.request.RequestID,
					Allowed:   m.selectedIndex != 2, // 0=Yes, 1=Yes+Remember, 2=No
					Remember:  m.selectedIndex == 1,
					Scope:     "project", // Save to project by default
				}
			}
		case "esc":
			// Cancel - treat as No
			return m, func() tea.Msg {
				return PermissionResponseMsg{
					RequestID: m.request.RequestID,
					Allowed:   false,
					Remember:  false,
				}
			}
		}
	}

	return m, nil
}

// View renders the permission dialog
func (m *PermissionDialogModel) View() string {
	if !m.visible || m.request == nil {
		return ""
	}

	// Dialog width
	dialogWidth := 80
	if m.width > 0 && m.width < dialogWidth {
		dialogWidth = m.width - 4
	}

	var content strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Theme.Primary).
		Render(fmt.Sprintf("%s %s", m.request.ToolName, m.getActionLabel()))

	hint := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Muted).
		Render("ctrl+e to explain")

	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		strings.Repeat(" ", dialogWidth-lipgloss.Width(title)-lipgloss.Width(hint)),
		hint,
	)
	content.WriteString(titleLine)
	content.WriteString("\n\n")

	// Command/Resource display
	commandBox := m.renderCommandBox(dialogWidth)
	content.WriteString(commandBox)
	content.WriteString("\n\n")

	// Question
	question := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Foreground).
		Render("Do you want to proceed?")
	content.WriteString(question)
	content.WriteString("\n")

	// Options
	options := m.renderOptions(dialogWidth)
	content.WriteString(options)
	content.WriteString("\n\n")

	// Help text
	help := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Muted).
		Render("Esc to cancel · Tab to add additional instructions")
	content.WriteString(help)

	// Add border
	dialogStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Theme.Primary).
		Padding(1, 2).
		Width(dialogWidth)

	dialog := dialogStyle.Render(content.String())

	// Center the dialog on screen
	if m.height > 0 {
		dialogHeight := lipgloss.Height(dialog)
		topPadding := (m.height - dialogHeight) / 2
		if topPadding > 0 {
			dialog = strings.Repeat("\n", topPadding) + dialog
		}
	}

	return dialog
}

// getActionLabel returns a human-readable action label
func (m *PermissionDialogModel) getActionLabel() string {
	toolName := strings.ToLower(m.request.ToolName)
	switch toolName {
	case "bash":
		return "command"
	case "read":
		return "read"
	case "write":
		return "write"
	case "edit":
		return "edit"
	default:
		return m.request.Action
	}
}

// renderCommandBox renders the command/resource display
func (m *PermissionDialogModel) renderCommandBox(width int) string {
	var lines []string

	// Main command/resource
	toolName := strings.ToLower(m.request.ToolName)
	if toolName == "bash" {
		if cmd, ok := m.request.Params["command"].(string); ok {
			cmdStyle := lipgloss.NewStyle().
				Foreground(m.styles.Theme.Foreground).
				Bold(true)
			lines = append(lines, cmdStyle.Render(cmd))
		}
	} else {
		resourceStyle := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Foreground).
			Bold(true)
		lines = append(lines, resourceStyle.Render(m.request.Resource))
	}

	// Description
	if m.request.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Muted)
		lines = append(lines, descStyle.Render(m.request.Description))
	}

	return strings.Join(lines, "\n")
}

// renderOptions renders the choice options
func (m *PermissionDialogModel) renderOptions(width int) string {
	options := []string{
		"Yes",
		m.getRememberOptionLabel(),
		"No",
	}

	var lines []string
	for i, option := range options {
		prefix := "  "
		if i == m.selectedIndex {
			prefix = lipgloss.NewStyle().
				Foreground(m.styles.Theme.Primary).
				Render("❯ ")
		}

		optionText := option
		if i == m.selectedIndex {
			optionText = lipgloss.NewStyle().
				Foreground(m.styles.Theme.Primary).
				Bold(true).
				Render(option)
		} else {
			optionText = lipgloss.NewStyle().
				Foreground(m.styles.Theme.Foreground).
				Render(option)
		}

		// Add number
		number := lipgloss.NewStyle().
			Foreground(m.styles.Theme.Muted).
			Render(fmt.Sprintf("%d. ", i+1))

		lines = append(lines, prefix+number+optionText)
	}

	return strings.Join(lines, "\n")
}

// getRememberOptionLabel returns the label for the "remember" option
func (m *PermissionDialogModel) getRememberOptionLabel() string {
	username := "user"
	// Try to get actual username from environment
	if u := getUserName(); u != "" {
		username = u
	}

	action := m.getActionLabel()
	toolName := strings.ToLower(m.request.ToolName)

	switch toolName {
	case "bash":
		// Extract command name
		if cmd, ok := m.request.Params["command"].(string); ok {
			cmdParts := strings.Fields(cmd)
			if len(cmdParts) > 0 {
				cmdName := cmdParts[0]
				return fmt.Sprintf("Yes, allow running %s from %s/ from this project", cmdName, username)
			}
		}
		return fmt.Sprintf("Yes, allow %s from %s/ from this project", action, username)
	case "read":
		return fmt.Sprintf("Yes, allow reading from %s/ from this project", username)
	case "write", "edit":
		return fmt.Sprintf("Yes, allow %s from %s/ from this project", action, username)
	default:
		return fmt.Sprintf("Yes, allow %s from %s/ from this project", action, username)
	}
}

// SetSize sets the dialog size
func (m *PermissionDialogModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// PermissionResponseMsg is sent when user responds to permission request
type PermissionResponseMsg struct {
	RequestID string
	Allowed   bool
	Remember  bool
	Scope     string // "project" or "global"
}
