package services

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/pages"
	svclib "main/internal/services"
	"main/internal/theme"
)

type Model struct {
	servicesList []svclib.Service
	cursor       int
}

func New() Model {
	return Model{cursor: 0}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.servicesList)-1 {
				m.cursor++
			}
		case "r", "R":
			if len(m.servicesList) > 0 {
				return m, func() tea.Msg {
					return pages.RunRemoteCmdMsg{Command: "sudo systemctl restart " + m.servicesList[m.cursor].Name}
				}
			}
		case "s", "S":
			if len(m.servicesList) > 0 {
				return m, func() tea.Msg {
					return pages.RunRemoteCmdMsg{Command: "sudo systemctl stop " + m.servicesList[m.cursor].Name}
				}
			}
		}

	case agent.Payload:
		m.servicesList = msg.Services
		return m, nil
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

	var items string
	if len(m.servicesList) == 0 {
		items = "No active services found."
	} else {
		for i, s := range m.servicesList {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			statusColor := theme.Current.Dim
			if s.Status == "active (running)" || strings.Contains(s.Status, "running") {
				statusColor = lipgloss.Color("46")
			} else if s.Status == "failed" {
				statusColor = lipgloss.Color("196")
			}

			items += fmt.Sprintf("%s [%s] %s\n", cursor, lipgloss.NewStyle().Foreground(statusColor).Render(s.Status), style.Render(s.Name))
		}
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [R] Restart Service  [S] Stop Service")

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("SYSTEMD SERVICES"),
			items,
			controls,
		),
	)
}

func (m Model) Title() string { return "Services" }
func (m Model) Icon() string { return "⚙️" }
