package services

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	svclib "main/internal/services"
)

type Model struct {
	servicesList []svclib.Service
}

func New() Model {
	return Model{
		servicesList: svclib.GetServices(), // Initialized instantly
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case agent.Payload:
		m.servicesList = msg.Services
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
		titleCard.Render("SYSTEM SERVICES (systemd)") + "\n" +
			svclib.FormatServices(m.servicesList),
	)
}

func (m Model) Title() string { return "Services" }
func (m Model) Icon() string { return "⚙️" }
