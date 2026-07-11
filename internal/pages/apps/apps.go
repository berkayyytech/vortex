package apps

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	appengine "main/internal/engine/apps"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	apps   []appengine.App
	cursor int
	engine *appengine.Engine
	status string
}

func New() Model {
	return Model{
		apps:   nil,
		cursor: 0,
		status: "Connecting to Application Engine...",
	}
}

func (m Model) Init() tea.Cmd { return nil }

type appsLoadedMsg []appengine.App

func fetchApps(engine *appengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		apps, err := engine.DetectApps()
		if err != nil {
			return nil // ignore silent failure
		}
		return appsLoadedMsg(apps)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case appsLoadedMsg:
		m.apps = msg
		m.status = fmt.Sprintf("Detected %d running applications.", len(m.apps))
		if m.cursor >= len(m.apps) {
			m.cursor = 0
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.apps)-1 {
				m.cursor++
			}
		case "s", "S":
			if len(m.apps) > 0 && m.engine != nil {
				return m, func() tea.Msg {
					m.engine.StopProcess(m.apps[m.cursor].PID)
					return fetchApps(m.engine)()
				}
			}
		case "K": // Shift+K for forced kill
			if len(m.apps) > 0 && m.engine != nil {
				return m, func() tea.Msg {
					m.engine.KillProcess(m.apps[m.cursor].PID)
					return fetchApps(m.engine)()
				}
			}
		case "r", "R":
			// Refresh
			m.status = "Scanning ports and runtimes..."
			return m, fetchApps(m.engine)
		}

	case sshlib.ConnectedMsg:
		m.engine = appengine.NewEngine(msg.Client)
		m.status = "Scanning ports and runtimes..."
		return m, fetchApps(m.engine)
	}
	return m, nil
}

func (m Model) View() string {
	var items string
	if m.apps == nil {
		items = lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.status)
	} else if len(m.apps) == 0 {
		items = "No active applications detected on listening ports."
	} else {
		for i, app := range m.apps {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			
			runtimeTag := lipgloss.NewStyle().Foreground(theme.Current.Accent).Render("[" + app.Runtime + "]")
			portTag := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("Port: " + app.Port)
			pidTag := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("PID: " + app.PID)
			
			items += fmt.Sprintf("%s %s %s  %s  %s\n", cursor, runtimeTag, style.Render(app.Name), portTag, pidTag)
		}
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [R] Refresh  [S] Stop  [Shift+K] Force Kill")

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("APPLICATION MANAGER"),
		items,
		controls,
	)

	return components.Card(content, 70)
}

func (m Model) Title() string { return "Apps" }
func (m Model) Icon() string { return "🚀" }
