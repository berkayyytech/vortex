package terminal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type OpenShellMsg struct{}

type Model struct{}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			return m, func() tea.Msg { return OpenShellMsg{} }
		}
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
			titleCard.Render("SECURE SHELL (SSH)"),
			"Host:     root@192.168.1.50",
			"Key:      ~/.ssh/id_rsa",
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("[ Press 'ENTER' to initiate SSH Session ]"),
		),
	)
}

func (m Model) Title() string { return "SSH" }
func (m Model) Icon() string { return "🔑" }
