package docker

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	"main/internal/config"
	docklib "main/internal/docker"
	dockerengine "main/internal/engine/docker"
	"main/internal/pages"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type OpenDockerShellMsg struct {
	ContainerID string
}

type Model struct {
	dockerStats docklib.DockerStats
	cursor      int
	engine      *dockerengine.Engine
	confirming  bool
	pendingAction string
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
		case "enter":
			if len(m.dockerStats.ContainersList) > 0 {
				c := m.dockerStats.ContainersList[m.cursor]
				return m, func() tea.Msg { return OpenDockerShellMsg{ContainerID: c.ID} }
			}
		case "r", "R":
			if len(m.dockerStats.ContainersList) > 0 && m.engine != nil {
				if config.CurrentConfig.Docker.ConfirmActions && !m.confirming {
					m.confirming = true
					m.pendingAction = "restart"
					return m, nil
				}
				c := m.dockerStats.ContainersList[m.cursor]
				m.confirming = false
				return m, func() tea.Msg {
					m.engine.RestartContainer(c.ID)
					return pages.LogActivityMsg{Message: "Restarted Docker container " + c.Name}
				}
			}
		case "s", "S":
			if len(m.dockerStats.ContainersList) > 0 && m.engine != nil {
				if config.CurrentConfig.Docker.ConfirmActions && !m.confirming {
					m.confirming = true
					m.pendingAction = "stop"
					return m, nil
				}
				c := m.dockerStats.ContainersList[m.cursor]
				m.confirming = false
				return m, func() tea.Msg {
					m.engine.StopContainer(c.ID)
					return pages.LogActivityMsg{Message: "Stopped Docker container " + c.Name}
				}
			}
		case "y", "Y":
			if m.confirming && len(m.dockerStats.ContainersList) > 0 && m.engine != nil {
				c := m.dockerStats.ContainersList[m.cursor]
				action := m.pendingAction
				m.confirming = false
				m.pendingAction = ""
				return m, func() tea.Msg {
					if action == "restart" {
						m.engine.RestartContainer(c.ID)
						return pages.LogActivityMsg{Message: "Restarted Docker container " + c.Name}
					}
					if action == "stop" {
						m.engine.StopContainer(c.ID)
						return pages.LogActivityMsg{Message: "Stopped Docker container " + c.Name}
					}
					return nil
				}
			}
		case "n", "N", "esc":
			if m.confirming {
				m.confirming = false
				m.pendingAction = ""
				return m, nil
			}
		}

	case sshlib.ConnectedMsg:
		m.engine = dockerengine.NewEngine(msg.Client)
		return m, nil

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

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [Enter] Connect  [R] Restart  [S] Stop")
	if m.confirming {
		controls = lipgloss.NewStyle().Foreground(theme.Current.Warning).Bold(true).Render(fmt.Sprintf("\nAre you sure you want to %s this container? (Y/N)", m.pendingAction))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("DOCKER ENGINE"),
		header,
		"\n"+list,
		controls,
	)

	return components.Card(content, 80)
}

func (m Model) Title() string { return "Docker" }
func (m Model) Icon() string { return "🐳" }
