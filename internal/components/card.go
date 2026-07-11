package components

import (
	"github.com/charmbracelet/lipgloss"
	"main/internal/theme"
)

// Card creates a standard bordered box for UI content.
func Card(content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Dim).
		Padding(1, 3).
		Margin(1, 0)
	
	if width > 0 {
		style = style.Width(width)
	}

	return style.Render(content)
}

// Title creates a standard header string for cards/pages.
func Title(title string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Accent).
		MarginBottom(1).
		Render(title)
}
