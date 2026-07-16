package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Layout holds the calculated dimensions for the various UI panes.
type Layout struct {
	HeaderHeight int
	TotalWidth   int
	TotalHeight  int

	ContentWidth  int
	ContentHeight int

	LeftWidth  int
	RightWidth int
}

// CalculateLayout uses standard Bubbletea/Lipgloss practices to compute
// the exact physical dimensions of the layout, cleanly separating the math
// from the UI component rendering.
func CalculateLayout(termWidth, termHeight int, headerStr string) Layout {
	var l Layout
	l.TotalWidth = termWidth
	l.TotalHeight = termHeight

	// If terminal is impossibly small, just return empty layout
	if termWidth < 10 || termHeight < 10 {
		return l
	}

	// Measure header height exactly
	hTitleFrame, _ := titleStyle.GetFrameSize()
	header := titleStyle.Width(termWidth - 1 - hTitleFrame).Align(lipgloss.Center).Render(headerStr)
	l.HeaderHeight = lipgloss.Height(header)

	// Determine available space for the main dashboard
	availHeight := termHeight - l.HeaderHeight

	// Extract the physical bounds of the outer dashboard frame
	hFrame, vFrame := outerStyle.GetFrameSize()

	// We subtract 1 from width and height as a standard terminal layout safety margin.
	l.ContentWidth = termWidth - hFrame - 1
	l.ContentHeight = availHeight - vFrame - 1

	l.LeftWidth = int(float64(l.ContentWidth) * 0.35)
	l.RightWidth = l.ContentWidth - l.LeftWidth

	return l
}
