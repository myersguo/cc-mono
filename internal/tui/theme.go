package tui

import "github.com/charmbracelet/lipgloss"

// Theme represents a color theme
type Theme struct {
	Name string

	// Colors
	Primary       lipgloss.Color
	Secondary     lipgloss.Color
	Accent        lipgloss.Color
	Background    lipgloss.Color
	Foreground    lipgloss.Color
	Muted         lipgloss.Color
	Success       lipgloss.Color
	Warning       lipgloss.Color
	Error         lipgloss.Color
	Border        lipgloss.Color
	UserMessage   lipgloss.Color
	AIMessage     lipgloss.Color
	ToolMessage   lipgloss.Color
	ThinkingColor lipgloss.Color
}

// DarkTheme returns the default dark theme (Claude Code style)
func DarkTheme() Theme {
	return Theme{
		Name:          "dark",
		Primary:       lipgloss.Color("#A78BFA"), // Light purple (Claude Code accent)
		Secondary:     lipgloss.Color("#60A5FA"), // Light blue
		Accent:        lipgloss.Color("#34D399"), // Emerald green
		Background:    lipgloss.Color("#0A0E14"), // Very dark blue-black (Claude Code bg)
		Foreground:    lipgloss.Color("#E5E7EB"), // Light gray text
		Muted:         lipgloss.Color("#6B7280"), // Medium gray
		Success:       lipgloss.Color("#34D399"), // Emerald green
		Warning:       lipgloss.Color("#FBBF24"), // Amber
		Error:         lipgloss.Color("#F87171"), // Light red
		Border:        lipgloss.Color("#1E3A8A"), // Deep blue (Claude Code border)
		UserMessage:   lipgloss.Color("#60A5FA"), // Light blue
		AIMessage:     lipgloss.Color("#A78BFA"), // Light purple
		ToolMessage:   lipgloss.Color("#34D399"), // Emerald green
		ThinkingColor: lipgloss.Color("#9CA3AF"), // Light gray
	}
}

// LightTheme returns the default light theme
func LightTheme() Theme {
	return Theme{
		Name:          "light",
		Primary:       lipgloss.Color("#7C3AED"), // Purple
		Secondary:     lipgloss.Color("#3B82F6"), // Blue
		Accent:        lipgloss.Color("#10B981"), // Green
		Background:    lipgloss.Color("#FFFFFF"), // White
		Foreground:    lipgloss.Color("#1F2937"), // Dark gray
		Muted:         lipgloss.Color("#6B7280"), // Medium gray
		Success:       lipgloss.Color("#10B981"), // Green
		Warning:       lipgloss.Color("#F59E0B"), // Orange
		Error:         lipgloss.Color("#EF4444"), // Red
		Border:        lipgloss.Color("#D1D5DB"), // Light gray
		UserMessage:   lipgloss.Color("#3B82F6"), // Blue
		AIMessage:     lipgloss.Color("#7C3AED"), // Purple
		ToolMessage:   lipgloss.Color("#10B981"), // Green
		ThinkingColor: lipgloss.Color("#9CA3AF"), // Light gray
	}
}

// Styles contains all the lipgloss styles for the TUI
type Styles struct {
	Theme Theme

	// Base styles
	App          lipgloss.Style
	Header       lipgloss.Style
	Footer       lipgloss.Style
	StatusBar    lipgloss.Style
	InputPrompt  lipgloss.Style
	InputCursor  lipgloss.Style
	Spinner      lipgloss.Style
	HelpKey      lipgloss.Style
	HelpValue    lipgloss.Style
	Error        lipgloss.Style
	Warning      lipgloss.Style
	Success      lipgloss.Style

	// Message styles
	UserMessage      lipgloss.Style
	AIMessage        lipgloss.Style
	ToolResultOk     lipgloss.Style
	ToolResultError  lipgloss.Style
	ThinkingMessage  lipgloss.Style
	MessageTimestamp lipgloss.Style
	MessageRole      lipgloss.Style

	// Component styles
	CodeBlock      lipgloss.Style
	InlineCode     lipgloss.Style
	Quote          lipgloss.Style
	List           lipgloss.Style
	Link           lipgloss.Style
	Separator      lipgloss.Style
	ToolName       lipgloss.Style
	ToolOutput     lipgloss.Style
	Markdown       lipgloss.Style

	// Border styles
	BorderNormal lipgloss.Style
	BorderActive lipgloss.Style
}

// NewStyles creates styles from a theme
func NewStyles(theme Theme) *Styles {
	s := &Styles{
		Theme: theme,
	}

	// Base styles
	s.App = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Background)

	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Primary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Border).
		Padding(0, 1)

	s.Footer = lipgloss.NewStyle().
		Foreground(theme.Muted).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(theme.Border).
		Padding(0, 1)

	s.StatusBar = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Primary).
		Padding(0, 1)

	s.InputPrompt = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	s.InputCursor = lipgloss.NewStyle().
		Foreground(theme.Accent)

	s.Spinner = lipgloss.NewStyle().
		Foreground(theme.Primary)

	s.HelpKey = lipgloss.NewStyle().
		Foreground(theme.Muted)

	s.HelpValue = lipgloss.NewStyle().
		Foreground(theme.Foreground)

	s.Error = lipgloss.NewStyle().
		Foreground(theme.Error).
		Bold(true)

	s.Warning = lipgloss.NewStyle().
		Foreground(theme.Warning).
		Bold(true)

	s.Success = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true)

	// Message styles (no borders)
	s.UserMessage = lipgloss.NewStyle().
		Foreground(theme.UserMessage).
		Padding(0, 1).
		MarginTop(1)

	s.AIMessage = lipgloss.NewStyle().
		Foreground(theme.AIMessage).
		Padding(0, 1).
		MarginTop(1)

	s.ToolResultOk = lipgloss.NewStyle().
		Foreground(theme.ToolMessage).
		Padding(0, 1).
		MarginTop(1)

	s.ToolResultError = lipgloss.NewStyle().
		Foreground(theme.Error).
		Padding(0, 1).
		MarginTop(1)

	s.ThinkingMessage = lipgloss.NewStyle().
		Foreground(theme.ThinkingColor).
		Italic(true).
		Padding(0, 1)

	s.MessageTimestamp = lipgloss.NewStyle().
		Foreground(theme.Muted).
		Italic(true)

	s.MessageRole = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true)

	// Component styles
	s.CodeBlock = lipgloss.NewStyle().
		Foreground(theme.Secondary).
		Background(lipgloss.Color("#1A1F2E")).
		Padding(1, 2).
		MarginTop(1).
		MarginBottom(1)

	s.InlineCode = lipgloss.NewStyle().
		Foreground(theme.Secondary).
		Background(lipgloss.Color("#1A1F2E")).
		Padding(0, 1)

	s.Quote = lipgloss.NewStyle().
		Foreground(theme.Muted).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(theme.Muted).
		PaddingLeft(2).
		MarginTop(1).
		MarginBottom(1)

	s.List = lipgloss.NewStyle().
		Foreground(theme.Foreground)

	s.Link = lipgloss.NewStyle().
		Foreground(theme.Accent).
		Underline(true)

	s.Separator = lipgloss.NewStyle().
		Foreground(theme.Border).
		MarginTop(1).
		MarginBottom(1)

	s.ToolName = lipgloss.NewStyle().
		Foreground(theme.ToolMessage).
		Bold(true)

	s.ToolOutput = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Padding(0, 2)

	s.Markdown = lipgloss.NewStyle().
		Foreground(theme.Foreground)

	// Border styles
	s.BorderNormal = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border)

	s.BorderActive = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary)

	return s
}

// GetTheme returns a theme by name
func GetTheme(name string) Theme {
	switch name {
	case "light":
		return LightTheme()
	case "dark":
		return DarkTheme()
	default:
		return DarkTheme()
	}
}
