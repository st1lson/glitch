package theme

import "github.com/charmbracelet/lipgloss"

var (
	// Base Colors
	ColorPrimary   = lipgloss.Color("6") // Cyan
	ColorSuccess   = lipgloss.Color("2") // Green
	ColorWarning   = lipgloss.Color("3") // Yellow
	ColorError     = lipgloss.Color("1") // Red
	ColorMuted     = lipgloss.Color("8") // Gray
	ColorText      = lipgloss.Color("7") // White
	ColorHighlight = lipgloss.Color("5") // Magenta

	// Base Styles
	Primary   = lipgloss.NewStyle().Foreground(ColorPrimary)
	Success   = lipgloss.NewStyle().Foreground(ColorSuccess)
	Warning   = lipgloss.NewStyle().Foreground(ColorWarning)
	Error     = lipgloss.NewStyle().Foreground(ColorError)
	Muted     = lipgloss.NewStyle().Foreground(ColorMuted)
	Text      = lipgloss.NewStyle().Foreground(ColorText)
	Highlight = lipgloss.NewStyle().Foreground(ColorHighlight)

	Bold = lipgloss.NewStyle().Bold(true)
)
