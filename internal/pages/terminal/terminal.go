package terminal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type OpenShellMsg struct{}

type Model struct {
	host string
}

func New() Model {
	return Model{host: "Not Connected"}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sshlib.ConnectedMsg:
		m.host = msg.User + "@" + msg.Host + ":" + msg.Port
		return m, nil
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
		BorderForeground(theme.Current.Dim).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Accent).
		MarginBottom(1)

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("SECURE SHELL (SSH)"),
			"Target:   "+lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.host),
			"",
			lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("[ Press 'ENTER' to initiate Native SSH Session ]"),
		),
	)
}

func (m Model) Title() string { return "SSH" }
func (m Model) Icon() string { return "🔑" }
