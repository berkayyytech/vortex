package services

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	sysengine "main/internal/engine/systemd"
	sshlib "main/internal/ssh"
	svclib "main/internal/services"
	"main/internal/theme"
)

type Model struct {
	servicesList []svclib.Service
	cursor       int
	engine       *sysengine.Engine
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
			if len(m.servicesList) > 0 && m.engine != nil {
				return m, func() tea.Msg {
					m.engine.RestartService(m.servicesList[m.cursor].Name)
					return nil
				}
			}
		case "s", "S":
			if len(m.servicesList) > 0 && m.engine != nil {
				return m, func() tea.Msg {
					m.engine.StopService(m.servicesList[m.cursor].Name)
					return nil
				}
			}
		}

	case sshlib.ConnectedMsg:
		m.engine = sysengine.NewEngine(msg.Client)
		return m, nil

	case agent.Payload:
		m.servicesList = msg.Services
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
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

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("SYSTEMD SERVICES"),
		items,
		controls,
	)

	return components.Card(content, 60)
}

func (m Model) Title() string { return "Services" }
func (m Model) Icon() string { return "⚙️" }
