package docker

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	docklib "main/internal/docker"
)

type Model struct {
	dockerStats docklib.DockerStats
}

func New() Model {
	return Model{
		dockerStats: docklib.DockerStats{Status: "Connecting..."},
	}
}

func (m Model) Init() tea.Cmd {
	return fetchDockerStatsCmd
}

type dockerMsg docklib.DockerStats

func fetchDockerStatsCmd() tea.Msg {
	return dockerMsg(docklib.GetDockerStats())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dockerMsg:
		m.dockerStats = docklib.DockerStats(msg)
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
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("DOCKER ENGINE API"),
			fmt.Sprintf("Status:             %s\n", m.dockerStats.Status),
			fmt.Sprintf("Active Containers:  %d", m.dockerStats.Containers),
			fmt.Sprintf("Images:             %d", m.dockerStats.Images),
			fmt.Sprintf("Networks:           %d", m.dockerStats.Networks),
			fmt.Sprintf("Volumes:            %d", m.dockerStats.Volumes),
		),
	)
}

func (m Model) Title() string { return "Docker" }
func (m Model) Icon() string { return "🐳" }
