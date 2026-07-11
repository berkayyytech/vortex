package components

import (
	"github.com/charmbracelet/lipgloss"
	"main/internal/theme"
	"time"
)

type Toast struct {
	Message   string
	Type      string // "info", "error", "success"
	ExpiresAt time.Time
	Active    bool
}

func NewToast(msg string, t string, duration time.Duration) Toast {
	return Toast{
		Message:   msg,
		Type:      t,
		ExpiresAt: time.Now().Add(duration),
		Active:    true,
	}
}

func (t *Toast) Update() {
	if t.Active && time.Now().After(t.ExpiresAt) {
		t.Active = false
	}
}

func (t *Toast) View() string {
	if !t.Active {
		return ""
	}

	color := theme.Current.Primary
	if t.Type == "error" {
		color = lipgloss.Color("196")
	} else if t.Type == "success" {
		color = lipgloss.Color("46")
	}

	style := lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color("0")). // black text for contrast
		Bold(true).
		Padding(0, 2).
		Margin(1, 0)

	return style.Render(t.Message)
}
