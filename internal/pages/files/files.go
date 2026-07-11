package files

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
)

type Model struct {
	filesData string
}

func New() Model {
	return Model{
		filesData: "Fetching remote file system...",
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case agent.Payload:
		if msg.Files != "" {
			m.filesData = msg.Files
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
			titleCard.Render("REMOTE FILE SYSTEM (/)"),
			m.filesData,
		),
	)
}

func (m Model) Title() string { return "Files" }
func (m Model) Icon() string { return "📁" }
