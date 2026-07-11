package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"main/internal/theme"
)

// ProgressBar returns a color-coded horizontal progress bar.
func ProgressBar(percent float64, totalBlocks int) string {
	filledBlocks := int((percent / 100.0) * float64(totalBlocks))
	if filledBlocks < 0 {
		filledBlocks = 0
	}
	if filledBlocks > totalBlocks {
		filledBlocks = totalBlocks
	}

	filled := strings.Repeat("█", filledBlocks)
	empty := strings.Repeat("░", totalBlocks-filledBlocks)

	var color lipgloss.Color
	if percent < 50 {
		color = lipgloss.Color("46") // Green
	} else if percent < 80 {
		color = lipgloss.Color("226") // Yellow
	} else {
		color = lipgloss.Color("196") // Red
	}

	bar := lipgloss.NewStyle().Foreground(color).Render(filled) +
		lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(empty) +
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf(" %3d%%", int(percent)))

	return bar
}
