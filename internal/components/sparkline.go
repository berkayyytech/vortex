package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var sparkChars = []string{" ", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

func Sparkline(data []float64, width int, color lipgloss.Color) string {
	if len(data) == 0 {
		return strings.Repeat(" ", width)
	}

	start := len(data) - width
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	pad := width - (len(data) - start)
	for i := 0; i < pad; i++ {
		sb.WriteString(" ")
	}

	for i := start; i < len(data); i++ {
		val := data[i]
		if val < 0 {
			val = 0
		} else if val > 100 {
			val = 100
		}
		
		idx := int((val / 100.0) * float64(len(sparkChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		sb.WriteString(sparkChars[idx])
	}

	return lipgloss.NewStyle().Foreground(color).Render(sb.String())
}
