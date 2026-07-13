package deploy

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	deployengine "main/internal/engine/deploy"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Mode int

const (
	ModeConfig Mode = iota
	ModeDeploying
	ModeDone
)

type Model struct {
	engine     *deployengine.Engine
	mode       Mode
	
	// Config fields
	appDir     string
	buildCmd   string
	restartCmd string
	healthUrl  string
	
	focusIndex int
	
	// Live output
	logs       []string
	outChan    <-chan string
}

func New() Model {
	return Model{
		mode:       ModeConfig,
		appDir:     "/var/www/myapp",
		buildCmd:   "npm install && npm run build",
		restartCmd: "pm2 restart myapp",
		healthUrl:  "http://localhost:3000/health",
		focusIndex: 0,
		logs:       []string{},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) IsInputActive() bool {
	return m.mode == ModeConfig && m.focusIndex < 4
}

type logMsg string
type deployDoneMsg struct{}

func waitForLogs(c <-chan string) tea.Cmd {
	return func() tea.Msg {
		if c == nil {
			return deployDoneMsg{}
		}
		msg, ok := <-c
		if !ok {
			return deployDoneMsg{}
		}
		return logMsg(msg)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sshlib.ConnectedMsg:
		m.engine = deployengine.NewEngine(msg.Client)
		// Optionally start webhook listener if we want webhooks triggered from the background
		// We use an arbitrary port for now
		if m.engine != nil {
			// This listens globally in the background
			m.engine.StartWebhookListener("8080", "/deploy", m.appDir, m.buildCmd, m.restartCmd, m.healthUrl)
		}
		return m, nil

	case logMsg:
		m.logs = append(m.logs, string(msg))
		// Keep scrolling limit if needed
		if len(m.logs) > 50 {
			m.logs = m.logs[len(m.logs)-50:]
		}
		return m, waitForLogs(m.outChan)

	case deployDoneMsg:
		m.mode = ModeDone
		return m, nil

	case tea.KeyMsg:
		if m.mode == ModeDeploying {
			// Ignore most keys while deploying, maybe allow cancellation in future
			return m, nil
		}

		if m.mode == ModeDone {
			switch msg.String() {
			case "enter", "esc", "r":
				m.mode = ModeConfig
				m.logs = []string{}
			}
			return m, nil
		}

		// ModeConfig
		switch msg.String() {
		case "up":
			if m.focusIndex > 0 {
				m.focusIndex--
			}
		case "down", "tab":
			if m.focusIndex < 4 {
				m.focusIndex++
			}
		case "enter":
			if m.focusIndex == 4 {
				// Trigger Deploy
				if m.engine != nil {
					m.mode = ModeDeploying
					m.logs = []string{"Starting deployment..."}
					outChan, err := m.engine.Deploy(m.appDir, m.buildCmd, m.restartCmd, m.healthUrl)
					if err != nil {
						m.logs = append(m.logs, "Error starting deployment: "+err.Error())
						m.mode = ModeDone
						return m, nil
					}
					m.outChan = outChan
					return m, waitForLogs(m.outChan)
				} else {
					m.logs = []string{"Engine not connected"}
					m.mode = ModeDone
				}
			} else {
				m.focusIndex++
			}
		case "backspace":
			m.updateCurrentField(func(s string) string {
				if len(s) > 0 {
					return s[:len(s)-1]
				}
				return s
			})
		case "esc":
			// exit focus? handled by main router maybe
		default:
			// Append typed character
			if len(msg.String()) == 1 {
				m.updateCurrentField(func(s string) string {
					return s + msg.String()
				})
			}
		}
	}

	return m, nil
}

func (m *Model) updateCurrentField(updater func(string) string) {
	switch m.focusIndex {
	case 0:
		m.appDir = updater(m.appDir)
	case 1:
		m.buildCmd = updater(m.buildCmd)
	case 2:
		m.restartCmd = updater(m.restartCmd)
	case 3:
		m.healthUrl = updater(m.healthUrl)
	}
}

func (m Model) View() string {
	if m.mode == ModeConfig {
		return m.renderConfig()
	}
	return m.renderLogs()
}

func (m Model) renderConfig() string {
	title := "DEPLOYMENT PIPELINE (v1.9)"
	
	activeStyle := lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(theme.Current.Text)
	dimStyle := lipgloss.NewStyle().Foreground(theme.Current.Dim)

	var items string
	
	renderField := func(idx int, label string, val string) string {
		cursor := "  "
		style := inactiveStyle
		if m.focusIndex == idx {
			cursor = "▶ "
			style = activeStyle
		}
		
		valDisplay := val
		if m.focusIndex == idx {
			valDisplay += "█"
		} else if val == "" {
			valDisplay = dimStyle.Render("(empty)")
		}
		
		return fmt.Sprintf("%s%-15s %s\n", cursor, style.Render(label+":"), style.Render(valDisplay))
	}

	items += renderField(0, "App Directory", m.appDir) + "\n"
	items += renderField(1, "Build Command", m.buildCmd) + "\n"
	items += renderField(2, "Restart Command", m.restartCmd) + "\n"
	items += renderField(3, "Health URL", m.healthUrl) + "\n"

	// Button
	btnCursor := "  "
	btnStyle := dimStyle
	if m.focusIndex == 4 {
		btnCursor = "▶ "
		btnStyle = lipgloss.NewStyle().Foreground(theme.Current.Text).Background(theme.Current.Success).Bold(true)
	}
	items += "\n" + btnCursor + btnStyle.Render(" [ DEPLOY NOW ] ") + "\n"
	
	items += "\n" + dimStyle.Render("Webhooks enabled: POST /deploy on port 8080 will trigger this config.")
	items += "\n" + dimStyle.Render("Logs are sent to Audit Log automatically.")

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title(title),
		"\n",
		items,
	)
	return components.Card(content, 110)
}

func (m Model) renderLogs() string {
	title := "DEPLOYMENT PIPELINE - RUNNING"
	if m.mode == ModeDone {
		title = "DEPLOYMENT PIPELINE - DONE (Press Enter to return)"
	}

	var logsDisplay string
	for _, l := range m.logs {
		if strings.Contains(l, "✖") || strings.Contains(l, "FAILED") {
			logsDisplay += lipgloss.NewStyle().Foreground(theme.Current.Error).Render(l) + "\n"
		} else if strings.Contains(l, "✔") || strings.Contains(l, "Success") {
			logsDisplay += lipgloss.NewStyle().Foreground(theme.Current.Success).Render(l) + "\n"
		} else {
			logsDisplay += lipgloss.NewStyle().Foreground(theme.Current.Text).Render(l) + "\n"
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title(title),
		"\n",
		logsDisplay,
	)
	return components.Card(content, 110)
}

func (m Model) Title() string { return "Deploy" }
func (m Model) Icon() string { return "🚀" }
