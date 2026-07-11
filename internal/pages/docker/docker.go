package docker

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	docklib "main/internal/docker"
	"main/internal/pages"
	"main/internal/theme"
)

type Model struct {
	dockerStats docklib.DockerStats
	cursor      int
}

func New() Model {
	return Model{
		dockerStats: docklib.DockerStats{Status: "Connecting..."},
		cursor:      0,
	}
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
			if m.cursor < len(m.dockerStats.ContainersList)-1 {
				m.cursor++
			}
		case "r", "R":
			if len(m.dockerStats.ContainersList) > 0 {
				c := m.dockerStats.ContainersList[m.cursor]
				return m, func() tea.Msg { return pages.RunRemoteCmdMsg{Command: "docker restart " + c.ID} }
			}
		case "s", "S":
			if len(m.dockerStats.ContainersList) > 0 {
				c := m.dockerStats.ContainersList[m.cursor]
				return m, func() tea.Msg { return pages.RunRemoteCmdMsg{Command: "docker stop " + c.ID} }
			}
		}

	case docklib.DockerStats:
		m.dockerStats = msg
		return m, nil
	case agent.Payload:
		m.dockerStats = msg.Docker
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

	header := fmt.Sprintf("Status: %s | Containers: %d | Images: %d | Volumes: %d",
		m.dockerStats.Status,
		m.dockerStats.Containers,
		m.dockerStats.Images,
		m.dockerStats.Volumes,
	)

	var list string
	if len(m.dockerStats.ContainersList) == 0 {
		list = "No containers found."
	} else {
		for i, c := range m.dockerStats.ContainersList {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			stateColor := theme.Current.Dim
			if c.State == "running" {
				stateColor = lipgloss.Color("46")
			} else if c.State == "exited" {
				stateColor = lipgloss.Color("196")
			}

			stateStr := lipgloss.NewStyle().Foreground(stateColor).Render(fmt.Sprintf("[%s]", c.State))
			list += fmt.Sprintf("%s %s %s %s\n", cursor, stateStr, style.Render(c.Name), lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(c.Image))
		}
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [R] Restart Container  [S] Stop Container")

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("DOCKER ENGINE"),
			header,
			"\n"+list,
			controls,
		),
	)
}

func (m Model) Title() string { return "Docker" }
func (m Model) Icon() string { return "🐳" }
