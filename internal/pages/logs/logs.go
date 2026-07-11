package logs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	logslib "main/internal/logs"
)

type Model struct {
	logsData string
}

func New() Model {
	return Model{
		logsData: logslib.GetRecentLogs(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case agent.Payload:
		if msg.Logs != "" {
			m.logsData = msg.Logs
		}
		return m, nil
	}
	return m, nil
}

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
			titleCard.Render("SYSTEM LOGS (/var/log/syslog)"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(m.logsData),
		),
	)
}

func (m Model) Title() string { return "Logs" }
func (m Model) Icon() string { return "📜" }
