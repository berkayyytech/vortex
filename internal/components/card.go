package components

import (
	"github.com/charmbracelet/lipgloss"
	"main/internal/config"
	"main/internal/theme"
)

func Card(content string, width int) string {
	bStyle := lipgloss.RoundedBorder()
	if config.CurrentConfig.Appearance.BorderStyle == "square" {
		bStyle = lipgloss.NormalBorder()
	}

	style := lipgloss.NewStyle().
		Border(bStyle).
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
