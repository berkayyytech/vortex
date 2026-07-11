package logs

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	logsengine "main/internal/engine/logs"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	logs   string
	engine *logsengine.Engine
	paused bool
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "p", "P":
			m.paused = !m.paused
			return m, nil
		}

	case sshlib.ConnectedMsg:
		m.engine = logsengine.NewEngine(msg.Client)
		return m, nil

	case agent.Payload:
		if !m.paused && m.engine != nil {
			// Fetch logs async
			return m, func() tea.Msg {
				out, err := m.engine.FetchSystemdLogs(30)
				if err != nil {
					return logsLoadedMsg("Error fetching logs: " + err.Error())
				}
				return logsLoadedMsg(out)
			}
		}
		return m, nil
	
	case logsLoadedMsg:
		m.logs = string(msg)
		return m, nil
	}
	return m, nil
}

type logsLoadedMsg string

func (m Model) View() string {
	var status string
	if m.paused {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("[PAUSED]")
	} else {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("[LIVE STREAMING]")
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [P] Pause/Resume Streaming")

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("SYSTEMD JOURNAL LOGS")+"  "+status,
		lipgloss.NewStyle().Foreground(theme.Current.Text).Render(m.logs),
		controls,
	)

	return components.Card(content, 100)
}

func (m Model) Title() string { return "Logs" }
func (m Model) Icon() string { return "📜" }
