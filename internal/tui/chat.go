package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/myersguo/cc-mono/pkg/agent"
	"github.com/myersguo/cc-mono/pkg/ai"
)

// ChatModel represents the main TUI model
type ChatModel struct {
	// Configuration
	styles     *Styles
	width      int
	height     int
	modelName  string
	provider   ai.Provider
	agent      *agent.Agent
	agentState *agent.AgentState
	agentCtx   *agent.AgentContext
	eventBus   *agent.EventBus
	ctx        context.Context
	cancel     context.CancelFunc

	// UI components
	viewport         viewport.Model
	editor           *Editor
	messageView      *MessageView
	spinner          spinner.Model
	permissionDialog *PermissionDialogModel
	historyManager   *HistoryManager

	// Hybrid rendering components
	useHybridMode        bool // Enable hybrid rendering mode
	lastRenderedIdx      int  // Index of last message rendered to stdout
	streamingContent     string
	needsOutputToStdout  []agent.AgentMessage // Messages pending output to stdout

	// State
	messages         []agent.AgentMessage
	streamingMessage *agent.AgentMessage // Current streaming message
	isAgentRunning   bool
	showWelcome      bool
	error            string
	statusMessage    string
	events           <-chan agent.AgentEvent
	workingDir       string

	// Config
	autoScroll bool
}

// NewChatModel creates a new chat model
func NewChatModel(
	agentInst *agent.Agent,
	themeName string,
) *ChatModel {
	theme := GetTheme(themeName)
	styles := NewStyles(theme)

	// Create viewport with initial height
	vp := viewport.New(80, 10)
	vp.Style = styles.App

	// Create history manager
	homeDir, _ := os.UserHomeDir()
	globalConfigDir := filepath.Join(homeDir, ".cc-mono")
	historyManager, err := NewHistoryManager(globalConfigDir, 1000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create history manager: %v\n", err)
		historyManager = nil
	}

	// Create editor with history manager
	editor := NewEditor(styles, "> ", historyManager)

	// Create message view
	messageView := NewMessageView(styles)

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	// Create permission dialog
	permDialog := NewPermissionDialog(styles)

	// Use existing agent state
	agentState := agentInst.GetState()

	// Use existing event bus
	eventBus := agentInst.GetEventBus()

	// Create permission manager
	projectDir := "."
	if cwd, err := os.Getwd(); err == nil {
		projectDir = cwd
	}
	permManager, err := agent.NewPermissionManager(globalConfigDir, projectDir)
	if err != nil {
		// Log error but continue without permission management
		fmt.Fprintf(os.Stderr, "Warning: Failed to create permission manager: %v\n", err)
		permManager = nil
	}

	// Create context with permission manager
	ctx, cancel := context.WithCancel(context.Background())
	if permManager != nil {
		ctx = context.WithValue(ctx, "permission_manager", permManager)
	}

	// Get working directory
	workingDir := "."
	if cwd, err := os.Getwd(); err == nil {
		workingDir = cwd
	}

	return &ChatModel{
		styles:              styles,
		viewport:            vp,
		editor:              editor,
		messageView:         messageView,
		spinner:             s,
		permissionDialog:    permDialog,
		historyManager:      historyManager,
		useHybridMode:       true, // Enable hybrid mode by default
		lastRenderedIdx:     -1,
		needsOutputToStdout: []agent.AgentMessage{},
		provider:            agentInst.GetProvider(),
		agent:               agentInst,
		agentState:          agentState,
		eventBus:            eventBus,
		ctx:                 ctx,
		cancel:              cancel,
		modelName:           agentState.GetModel().Name,
		messages:            []agent.AgentMessage{},
		autoScroll:          true,
		showWelcome:         true,
		workingDir:          workingDir,
	}
}

// Init initializes the model
func (m *ChatModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.listenForEvents(),
		m.editor.Focus(), // Always focus editor on start
	)
}

// Update handles messages
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case PermissionResponseMsg:
		// User responded to permission request
		if pm := m.ctx.Value("permission_manager"); pm != nil {
			permManager := pm.(*agent.PermissionManager)
			permManager.RespondToRequest(msg.RequestID, msg.Allowed, msg.Remember, msg.Scope)
		}
		m.permissionDialog.Hide()
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update permission dialog size
		m.permissionDialog.SetSize(msg.Width, msg.Height)

		// Update editor width first
		m.editor.SetWidth(msg.Width)

		// Calculate component heights
		headerHeight := lipgloss.Height(m.renderHeader())
		footerHeight := lipgloss.Height(m.renderFooter())
		editorHeight := lipgloss.Height(m.editor.View())

		// Calculate max available height for viewport
		maxViewportHeight := msg.Height - headerHeight - footerHeight - editorHeight

		// Set viewport width
		m.viewport.Width = msg.Width

		// Calculate actual content height
		contentLines := m.viewport.TotalLineCount()
		if contentLines == 0 {
			// No content yet, use default
			contentLines = 10
		}

		// Use dynamic height: min=10, max=available space
		minHeight := 10
		desiredHeight := contentLines
		if desiredHeight < minHeight {
			desiredHeight = minHeight
		}
		if desiredHeight > maxViewportHeight {
			desiredHeight = maxViewportHeight
		}

		m.viewport.Height = desiredHeight

		// Update content
		m.updateViewportContent()

	case tea.KeyMsg:
		// If permission dialog is visible, let it handle the key first
		if m.permissionDialog.IsVisible() {
			permDialog, permCmd := m.permissionDialog.Update(msg)
			m.permissionDialog = permDialog
			if permCmd != nil {
				cmds = append(cmds, permCmd)
			}
			// Don't process other keys when dialog is showing
			return m, tea.Batch(cmds...)
		}

		// Handle global keys first
		switch msg.String() {
		case "ctrl+c":
			// Quit on Ctrl+C
			m.cancel()
			// Flush history to disk before quitting
			if m.historyManager != nil {
				m.historyManager.Flush()
			}
			return m, tea.Quit

		case "ctrl+r":
			// Regenerate last response
			return m, m.regenerateLastResponse()

		case "ctrl+k", "pageup":
			m.viewport.LineUp(1)
			m.autoScroll = false

		case "ctrl+j", "pagedown":
			m.viewport.LineDown(1)
			// Re-enable auto-scroll if at bottom
			if m.viewport.AtBottom() {
				m.autoScroll = true
			}

		case "ctrl+u":
			m.viewport.HalfViewUp()
			m.autoScroll = false

		case "ctrl+d":
			m.viewport.HalfViewDown()
			if m.viewport.AtBottom() {
				m.autoScroll = true
			}

		default:
			// Hide welcome screen on first keypress
			if m.showWelcome {
				m.showWelcome = false
			}
			// Let editor handle all other keys
			var editorCmd tea.Cmd
			m.editor, editorCmd = m.editor.Update(msg)
			return m, editorCmd
		}

	case EditorSubmitMsg:
		// User submitted a message
		m.editor.Reset()

		// Hide welcome screen once user starts chatting
		m.showWelcome = false

		// Add user message
		userMsg := ai.UserMessage{
			Type:      ai.MessageTypeUser,
			Content:   []ai.Content{ai.NewTextContent(msg.Content)},
			Timestamp: time.Now().UnixMilli(),
		}

		agentMsg := agent.NewAgentMessage(
			userMsg,
			fmt.Sprintf("user-%d", time.Now().UnixNano()),
			time.Now().UnixMilli(),
		)

		m.messages = append(m.messages, agentMsg)
		m.agentState.AddMessage(agentMsg)
		m.autoScroll = true // Enable auto-scroll for new user message
		m.updateViewportContent()

		// Start agent loop
		return m, m.startAgent([]agent.AgentMessage{agentMsg})

	case EditorCancelMsg:
		// User cancelled editing (Esc key)
		m.editor.Reset()
		return m, nil

	case agent.AgentEvent:
		// Handle agent events
		model, cmd := m.handleAgentEvent(msg)
		// Continue listening for more events
		return model, tea.Batch(cmd, m.listenForEvents())

	case AgentEventMsg:
		// Received event from listener
		model, cmd := m.handleAgentEvent(msg.Event)
		// Continue listening for more events
		return model, tea.Batch(cmd, m.listenForEvents())

	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		return m, spinnerCmd

	case RefreshMsg:
		// Just trigger a re-render, no state change needed
		return m, nil
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// In hybrid mode, output pending messages using tea.Println
	if m.useHybridMode && len(m.needsOutputToStdout) > 0 {
		for _, msg := range m.needsOutputToStdout {
			rendered := m.messageView.Render(msg, m.width)
			cmds = append(cmds, tea.Println(rendered))
		}
		// Clear the queue
		m.needsOutputToStdout = []agent.AgentMessage{}

		// Clear streaming content after output to prevent view shrinking
		m.streamingContent = ""

		// Force a refresh to reposition the TUI after println
		cmds = append(cmds, func() tea.Msg {
			return RefreshMsg{}
		})
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m *ChatModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Show welcome screen if no messages yet
	if m.showWelcome && len(m.messages) == 0 {
		return m.renderWelcomeScreen()
	}

	// If permission dialog is visible, show it as overlay
	if m.permissionDialog.IsVisible() {
		// Just render the dialog, it will be centered
		return m.permissionDialog.View()
	}

	// In hybrid mode, only render streaming content (if any), editor and footer
	// All completed messages are output to stdout via tea.Println
	if m.useHybridMode {
		sections := []string{}

		// Show streaming content if present
		if m.streamingContent != "" {
			sections = append(sections, m.streamingContent)
		}

		// Always show editor and footer
		sections = append(sections, m.editor.View(), m.renderFooter())

		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	// Legacy viewport mode - render full UI
	sections := []string{
		m.renderHeader(),
		m.viewport.View(),
		m.editor.View(),
		m.renderFooter(),
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header
func (m *ChatModel) renderHeader() string {
	title := fmt.Sprintf("CC-Mono Chat - %s", m.modelName)
	if m.isAgentRunning {
		title += " " + m.spinner.View()
	}

	return m.styles.Header.Width(m.width).Render(title)
}

// renderFooter renders the footer
func (m *ChatModel) renderFooter() string {
	var parts []string

	if m.error != "" {
		parts = append(parts, m.styles.Error.Render("Error: "+m.error))
	} else if m.statusMessage != "" {
		parts = append(parts, m.styles.HelpValue.Render(m.statusMessage))
	}

	// Help text
	help := []string{
		m.styles.HelpKey.Render("Ctrl+C") + m.styles.HelpValue.Render(" quit"),
		m.styles.HelpKey.Render("Ctrl+K/J") + m.styles.HelpValue.Render(" scroll"),
		m.styles.HelpKey.Render("Ctrl+R") + m.styles.HelpValue.Render(" regenerate"),
	}
	parts = append(parts, strings.Join(help, " • "))

	footer := strings.Join(parts, " | ")
	return m.styles.Footer.Width(m.width).Render(footer)
}

// updateViewportContent updates the viewport content
// Delegates to hybrid or viewport mode based on configuration
func (m *ChatModel) updateViewportContent() {
	if m.useHybridMode {
		m.updateHybridMode()
	} else {
		m.updateViewportMode()
	}
}

// updateHybridMode updates content using hybrid rendering
// Marks messages that need to be output to stdout via tea.Println()
func (m *ChatModel) updateHybridMode() {
	// Mark new completed messages for output
	for i := m.lastRenderedIdx + 1; i < len(m.messages); i++ {
		msg := m.messages[i]

		// Only mark if this is not the currently streaming message
		if m.streamingMessage == nil || msg.ID != m.streamingMessage.ID {
			m.needsOutputToStdout = append(m.needsOutputToStdout, msg)
			m.lastRenderedIdx = i
		}
	}

	// Update streaming content for View()
	if m.streamingMessage != nil {
		// Actively streaming - render current streaming message
		m.streamingContent = m.messageView.Render(*m.streamingMessage, m.width)
	} else if len(m.needsOutputToStdout) > 0 {
		// Streaming complete, but not yet output - render the completed message(s)
		// This keeps the view height stable until tea.Println outputs
		var content strings.Builder
		for _, msg := range m.needsOutputToStdout {
			rendered := m.messageView.Render(msg, m.width)
			content.WriteString(rendered)
			if len(m.needsOutputToStdout) > 1 {
				content.WriteString("\n")
			}
		}
		m.streamingContent = content.String()
	} else {
		// Nothing to display
		m.streamingContent = ""
	}

	// Clear viewport content in hybrid mode
	m.viewport.SetContent("")
}

// updateViewportMode updates content using viewport rendering (legacy mode)
// All messages are rendered in the viewport
func (m *ChatModel) updateViewportMode() {
	var content strings.Builder

	// Render all completed messages
	for _, msg := range m.messages {
		rendered := m.messageView.Render(msg, m.viewport.Width)
		content.WriteString(rendered)
		content.WriteString("\n")
	}

	// Render streaming message if present
	if m.streamingMessage != nil {
		rendered := m.messageView.Render(*m.streamingMessage, m.viewport.Width)
		content.WriteString(rendered)
		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())

	// Adjust viewport height based on content
	m.adjustViewportHeight()

	// Auto-scroll to bottom if enabled
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

// adjustViewportHeight adjusts the viewport height based on content
func (m *ChatModel) adjustViewportHeight() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Calculate component heights
	headerHeight := lipgloss.Height(m.renderHeader())
	footerHeight := lipgloss.Height(m.renderFooter())
	editorHeight := lipgloss.Height(m.editor.View())

	// Calculate max available height for viewport
	maxViewportHeight := m.height - headerHeight - footerHeight - editorHeight

	// Get actual content height (total lines in viewport)
	contentLines := m.viewport.TotalLineCount()
	if contentLines == 0 {
		contentLines = 10 // Default to 10 when no content
	}

	// Use dynamic height: min=10, max=available space
	minHeight := 10
	desiredHeight := contentLines
	if desiredHeight < minHeight {
		desiredHeight = minHeight
	}
	if desiredHeight > maxViewportHeight {
		desiredHeight = maxViewportHeight
	}

	m.viewport.Height = desiredHeight
}

// handleAgentEvent handles agent events
func (m *ChatModel) handleAgentEvent(event agent.AgentEvent) (tea.Model, tea.Cmd) {
	switch e := event.(type) {
	case agent.AgentStartEvent:
		m.isAgentRunning = true
		m.statusMessage = "Agent started..."
		m.autoScroll = true // Enable auto-scroll for streaming
		m.updateViewportContent()

	case agent.AgentEndEvent:
		m.isAgentRunning = false
		m.statusMessage = "Agent completed"
		m.messages = e.Messages
		// Reset render tracking since messages were replaced
		if m.useHybridMode {
			m.lastRenderedIdx = -1
		}
		m.updateViewportContent()

	case agent.TurnStartEvent:
		m.statusMessage = "Processing..."
		// Clear any previous streaming message
		m.streamingMessage = nil

	case agent.TurnEndEvent:
		// Clear streaming message
		m.streamingMessage = nil

		// Add final assistant message
		m.messages = append(m.messages, e.Message)

		// Add tool results
		for _, toolResult := range e.ToolResults {
			toolMsg := agent.NewAgentMessage(
				toolResult,
				fmt.Sprintf("tool-%d", time.Now().UnixNano()),
				time.Now().UnixMilli(),
			)
			m.messages = append(m.messages, toolMsg)
		}

		m.autoScroll = true // Enable auto-scroll
		m.updateViewportContent()

	case agent.PromptAddedEvent:
		// Check if message is already in our local list to avoid duplicates
		exists := false
		for _, msg := range m.messages {
			if msg.ID == e.Message.ID {
				exists = true
				break
			}
		}
		if !exists {
			m.messages = append(m.messages, e.Message)
			m.autoScroll = true
			m.updateViewportContent()
		}

	case agent.MessageUpdateEvent:
		// Update streaming message
		m.statusMessage = "Streaming response..."
		m.streamingMessage = &e.Message
		m.autoScroll = true // Enable auto-scroll for streaming
		m.updateViewportContent()

	case agent.ToolExecutionStartEvent:
		m.statusMessage = fmt.Sprintf("Running tool: %s", e.ToolName)

		// Create and display a tool call message (without "You" header)
		toolCallText := m.messageView.RenderToolCallDisplay(e.ToolName, e.Args)

		// Create a special assistant message for tool call display
		toolCallMsg := agent.AgentMessage{
			Message: ai.AssistantMessage{
				Type: ai.MessageTypeAssistant,
				Content: []ai.Content{
					ai.NewTextContent(toolCallText),
				},
				Model:     "", // Empty model name to avoid showing role header
				Timestamp: time.Now().UnixMilli(),
			},
			ID:        fmt.Sprintf("toolcall-%s", e.ToolName),
			CreatedAt: time.Now().UnixMilli(),
		}
		m.messages = append(m.messages, toolCallMsg)
		m.autoScroll = true // Enable auto-scroll
		m.updateViewportContent()

	case agent.ToolExecutionEndEvent:
		if e.IsError {
			m.statusMessage = fmt.Sprintf("Tool %s failed", e.ToolName)
		} else {
			m.statusMessage = fmt.Sprintf("Tool %s completed", e.ToolName)
		}

	case agent.PermissionRequestEvent:
		// Show permission request dialog
		m.statusMessage = fmt.Sprintf("Permission required: %s", e.Request.Description)

		// Auto-approve safe operations (Read-only commands)
		if e.Request.RiskLevel == "safe" {
			if pm := m.ctx.Value("permission_manager"); pm != nil {
				permManager := pm.(*agent.PermissionManager)
				permManager.RespondToRequest(e.Request.RequestID, true, false, "project")
			}
		} else {
			// For medium/dangerous operations, show dialog
			m.permissionDialog.Show(e.Request)
		}

	case agent.ErrorEvent:
		m.isAgentRunning = false
		if e.Context != "" {
			m.error = fmt.Sprintf("%s: %s", e.Context, e.Error)
			// Print error to stderr for debugging
			fmt.Fprintf(os.Stderr, "[ERROR] %s: %s\n", e.Context, e.Error)
		} else {
			m.error = e.Error
			fmt.Fprintf(os.Stderr, "[ERROR] %s\n", e.Error)
		}
		m.statusMessage = ""
	}

	return m, nil
}

// startAgent starts the agent loop
func (m *ChatModel) startAgent(prompts []agent.AgentMessage) tea.Cmd {
	return func() tea.Msg {
		// Create agent context (using the shared agent)
		m.agentCtx = agent.NewAgentContext(m.agent)

		// Start agent loop in background
		go func() {
			config := &agent.AgentLoopConfig{
				MaxTurns:       10,
				MaxToolCalls:   5,
				EnableSteering: true,
			}

			err := agent.AgentLoop(m.ctx, prompts, m.agentCtx, config, m.eventBus)
			if err != nil {
				m.eventBus.Publish(agent.NewErrorEvent(err, "agent loop"))
			}
		}()

		return nil
	}
}

// regenerateLastResponse regenerates the last assistant response
func (m *ChatModel) regenerateLastResponse() tea.Cmd {
	// Find last user message
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Message.GetType() == ai.MessageTypeUser {
			// Remove all messages after this user message
			m.messages = m.messages[:i+1]
			m.updateViewportContent()

			// Restart agent
			return m.startAgent([]agent.AgentMessage{m.messages[i]})
		}
	}

	return nil
}

// listenForEvents listens for agent events
func (m *ChatModel) listenForEvents() tea.Cmd {
	// Subscribe to events
	m.events = m.eventBus.Subscribe(100)

	return func() tea.Msg {
		// This will block until an event is received
		event := <-m.events
		return AgentEventMsg{Event: event}
	}
}

// AgentEventMsg wraps an agent event for Bubbletea
type AgentEventMsg struct {
	Event agent.AgentEvent
}

// RefreshMsg triggers a view refresh without any state change
type RefreshMsg struct{}

// renderWelcomeScreen renders the welcome screen
func (m *ChatModel) renderWelcomeScreen() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Get username (or use default)
	username := os.Getenv("USER")
	if username == "" {
		username = "user"
	}

	// In hybrid mode, output simplified welcome to stdout and hide it
	if m.useHybridMode {
		welcomeMsg := fmt.Sprintf("\n✨ Welcome back %s!\n", username)
		welcomeMsg += fmt.Sprintf("   Model: %s\n", m.modelName)
		welcomeMsg += fmt.Sprintf("   Working directory: %s\n\n", m.workingDir)
		fmt.Print(welcomeMsg)

		// Hide welcome after first render in hybrid mode
		m.showWelcome = false

		// Return minimal UI
		sections := []string{
			m.editor.View(),
			m.renderFooter(),
		}
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	// Limit welcome screen height to about 15 lines
	contentHeight := 15

	// Left panel - Welcome and logo
	leftPanel := lipgloss.NewStyle().
		Width(m.width/2).
		Height(contentHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Theme.Primary).
		Padding(2, 4)

	// Welcome message
	welcomeText := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Theme.Foreground).
		Render(fmt.Sprintf("Welcome back %s!", username))

	// ASCII logo (simple)
	logo := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Primary).
		Render(`
    ╔═╗╔═╗  ╔╦╗╔═╗╔╗╔╔═╗
    ║  ║     ║║║║ ║║║║║ ║
    ╚═╝╚═╝  ╩ ╩╚═╝╝╚╝╚═╝
`)

	// Model and working dir info
	modelInfo := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Muted).
		Render(fmt.Sprintf("%s\n%s", m.modelName, m.workingDir))

	leftContent := lipgloss.JoinVertical(
		lipgloss.Left,
		welcomeText,
		"",
		logo,
		"",
		modelInfo,
	)

	// Right panel - Tips and recent activity
	rightPanel := lipgloss.NewStyle().
		Width(m.width/2).
		Height(contentHeight).
		Padding(2, 4)

	// Tips section
	tipsTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Theme.Accent).
		Render("Tips for getting started")

	tipsContent := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Foreground).
		Render("• Just start typing to begin\n• Use Ctrl+J to insert a new line\n• Press Enter to send your message\n• Use Esc to clear input")

	// Recent activity section
	activityTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Theme.Accent).
		Render("Recent activity")

	activityContent := lipgloss.NewStyle().
		Foreground(m.styles.Theme.Muted).
		Render("No recent activity")

	rightContent := lipgloss.JoinVertical(
		lipgloss.Left,
		tipsTitle,
		tipsContent,
		"",
		"",
		activityTitle,
		activityContent,
	)

	// Combine panels
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPanel.Render(leftContent),
		rightPanel.Render(rightContent),
	)

	// Build the complete view with editor and footer
	sections := []string{
		mainContent,
		m.editor.View(),
		m.renderFooter(),
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
