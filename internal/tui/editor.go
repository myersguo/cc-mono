package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Editor represents a multi-line text input editor
type Editor struct {
	textarea       textarea.Model
	styles         *Styles
	prompt         string
	focused        bool
	minHeight      int
	maxHeight      int
	historyManager *HistoryManager // Persistent history manager
	historyCache   []string        // Cached history for quick access
	historyIndex   int             // Current position in history (-1 means not browsing)
	currentDraft   string          // Current draft when browsing history
}

// NewEditor creates a new editor
func NewEditor(styles *Styles, prompt string, historyManager *HistoryManager) *Editor {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Ctrl+J for new line, Esc to clear)"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // No limit
	ta.SetHeight(1)  // Start with single line

	// Disable default Enter key handling in textarea
	ta.KeyMap.InsertNewline.SetEnabled(false)

	// Set styles
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = styles.InputPrompt
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(styles.Theme.Foreground)
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(styles.Theme.Muted)
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(styles.Theme.Muted)

	// Load history from manager
	var historyCache []string
	if historyManager != nil {
		historyCache = historyManager.GetAll()
	} else {
		historyCache = make([]string, 0)
	}

	return &Editor{
		textarea:       ta,
		styles:         styles,
		prompt:         prompt,
		focused:        false,
		minHeight:      1,
		maxHeight:      10,
		historyManager: historyManager,
		historyCache:   historyCache,
		historyIndex:   -1,
	}
}

// Init initializes the editor
func (e *Editor) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (e *Editor) Update(msg tea.Msg) (*Editor, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Submit message on Enter
			if e.focused && strings.TrimSpace(e.textarea.Value()) != "" {
				content := e.textarea.Value()
				// Add to history
				e.AddToHistory(content)
				return e, func() tea.Msg {
					return EditorSubmitMsg{Content: content}
				}
			}
		case tea.KeyCtrlJ:
			// Insert newline on Ctrl+J
			if e.focused {
				e.textarea.InsertString("\n")
				// Adjust height
				lines := strings.Count(e.textarea.Value(), "\n") + 1
				if lines < e.minHeight {
					e.textarea.SetHeight(e.minHeight)
				} else if lines > e.maxHeight {
					e.textarea.SetHeight(e.maxHeight)
				} else {
					e.textarea.SetHeight(lines)
				}
				return e, nil
			}
		case tea.KeyEsc:
			// Clear input on Esc
			if e.focused && e.textarea.Value() != "" {
				e.textarea.Reset()
				e.historyIndex = -1
				e.currentDraft = ""
				return e, nil
			}
		case tea.KeyUp:
			// Navigate up in history
			if e.focused && len(e.historyCache) > 0 {
				// Only single-line input can use arrow keys for history
				if strings.Count(e.textarea.Value(), "\n") == 0 {
					if e.historyIndex == -1 {
						// Save current draft
						e.currentDraft = e.textarea.Value()
						e.historyIndex = len(e.historyCache) - 1
					} else if e.historyIndex > 0 {
						e.historyIndex--
					}
					e.textarea.SetValue(e.historyCache[e.historyIndex])
					e.textarea.CursorEnd()
					return e, nil
				}
			}
		case tea.KeyDown:
			// Navigate down in history
			if e.focused && e.historyIndex != -1 {
				// Only single-line input can use arrow keys for history
				if strings.Count(e.textarea.Value(), "\n") == 0 {
					if e.historyIndex < len(e.historyCache)-1 {
						e.historyIndex++
						e.textarea.SetValue(e.historyCache[e.historyIndex])
					} else {
						// Restore draft
						e.historyIndex = -1
						e.textarea.SetValue(e.currentDraft)
					}
					e.textarea.CursorEnd()
					return e, nil
				}
			}
		}
	}

	// Update textarea (will not handle Enter anymore since we disabled it)
	e.textarea, cmd = e.textarea.Update(msg)

	// Adjust height based on content
	lines := strings.Count(e.textarea.Value(), "\n") + 1
	if lines < e.minHeight {
		e.textarea.SetHeight(e.minHeight)
	} else if lines > e.maxHeight {
		e.textarea.SetHeight(e.maxHeight)
	} else {
		e.textarea.SetHeight(lines)
	}

	return e, cmd
}

// View renders the editor
func (e *Editor) View() string {
	if !e.focused {
		return ""
	}

	// Render textarea
	view := e.textarea.View()

	// Add border
	bordered := e.styles.BorderActive.
		Width(e.textarea.Width()).
		Render(view)

	// Add help text
	help := e.styles.HelpKey.Render("Enter") +
		e.styles.HelpValue.Render(" send") +
		e.styles.HelpKey.Render(" • Ctrl+J") +
		e.styles.HelpValue.Render(" new line") +
		e.styles.HelpKey.Render(" • Esc") +
		e.styles.HelpValue.Render(" clear")

	return lipgloss.JoinVertical(lipgloss.Left, bordered, help)
}

// Focus focuses the editor
func (e *Editor) Focus() tea.Cmd {
	e.focused = true
	return e.textarea.Focus()
}

// Blur removes focus from the editor
func (e *Editor) Blur() {
	e.focused = false
	e.textarea.Blur()
}

// SetWidth sets the editor width
func (e *Editor) SetWidth(width int) {
	e.textarea.SetWidth(width - 4) // Account for border and padding
}

// SetHeight sets the editor height constraints
func (e *Editor) SetHeight(min, max int) {
	e.minHeight = min
	e.maxHeight = max
}

// Value returns the current editor value
func (e *Editor) Value() string {
	return e.textarea.Value()
}

// Reset resets the editor
func (e *Editor) Reset() {
	e.textarea.Reset()
	e.historyIndex = -1
	e.currentDraft = ""
}

// AddToHistory adds a message to the input history
func (e *Editor) AddToHistory(content string) {
	// Add to history manager (in-memory only, will be flushed on exit)
	if e.historyManager != nil {
		e.historyManager.Add(content)
		// Refresh cache from manager
		e.historyCache = e.historyManager.GetAll()
	} else {
		// Fallback to in-memory only (no persistence)
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		if len(e.historyCache) > 0 && e.historyCache[len(e.historyCache)-1] == content {
			return
		}
		e.historyCache = append(e.historyCache, content)
		if len(e.historyCache) > 100 {
			e.historyCache = e.historyCache[1:]
		}
	}

	// Reset history browsing state
	e.historyIndex = -1
	e.currentDraft = ""
}

// IsFocused returns whether the editor is focused
func (e *Editor) IsFocused() bool {
	return e.focused
}

// EditorSubmitMsg is sent when the user submits the editor
type EditorSubmitMsg struct {
	Content string
}

// EditorCancelMsg is sent when the user cancels the editor
type EditorCancelMsg struct{}
