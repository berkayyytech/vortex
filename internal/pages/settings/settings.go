package settings

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct{}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m Model) View() string {
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("CONFIGURATION"),
			"[x] Auto-refresh metrics (5s)",
			"[x] Enable Dark Mode",
			"[ ] Cache Docker credentials",
			"[ ] Save SSH keys locally",
		),
	)
}

func (m Model) Title() string { return "Settings" }
func (m Model) Icon() string { return "🛠️" }
