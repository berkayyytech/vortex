package alerts

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	"main/internal/config"
	"main/internal/engine/alerts"
	"main/internal/pages"
	"main/internal/theme"
)

type mode int

const (
	modeList mode = iota
	modeAdd
)

type Model struct {
	engine      *alerts.Engine
	webhooks    []alerts.Webhook
	cursor      int
	currentMode mode
	inputs      []textinput.Model
	focusIndex  int
	status      string
	width       int
	height      int
}

func init() {
	pages.Register(New())
}

func New() Model {
	eng := alerts.NewEngine()
	inputs := make([]textinput.Model, 3)
	
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "My Alert"
	inputs[0].Focus()
	
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "discord or slack"
	
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "https://..."

	return Model{
		engine:      eng,
		webhooks:    eng.GetWebhooks(),
		currentMode: modeList,
		inputs:      inputs,
		status:      "Idle",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case agent.Payload:
		if m.engine != nil {
			m.engine.CheckThresholds(msg.Stats, config.CurrentConfig.Monitoring)
			m.webhooks = m.engine.GetWebhooks()
		}
		return m, nil

	case tea.KeyMsg:
		if m.currentMode == modeAdd {
			switch msg.String() {
			case "esc":
				m.currentMode = modeList
				return m, nil
			case "enter":
				if m.focusIndex == len(m.inputs) {
					wType := alerts.WebhookType(strings.ToLower(m.inputs[1].Value()))
					if wType != alerts.Discord && wType != alerts.Slack {
						m.status = "❌ Type must be 'discord' or 'slack'"
						return m, nil
					}

					w := alerts.Webhook{
						Name:    m.inputs[0].Value(),
						Type:    wType,
						URL:     m.inputs[2].Value(),
						Enabled: true,
					}
					err := m.engine.AddWebhook(w)
					if err != nil {
						m.status = "❌ Error: " + err.Error()
					} else {
						m.status = "✅ Webhook added!"
						m.currentMode = modeList
						m.webhooks = m.engine.GetWebhooks()
					}
					return m, nil
				}
			case "up", "shift+tab":
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}
				cmds = append(cmds, m.updateFocus()...)
				return m, tea.Batch(cmds...)
			case "down", "tab":
				m.focusIndex++
				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				}
				cmds = append(cmds, m.updateFocus()...)
				return m, tea.Batch(cmds...)
			}

			for i := range m.inputs {
				if i == m.focusIndex {
					var cmd tea.Cmd
					m.inputs[i], cmd = m.inputs[i].Update(msg)
					cmds = append(cmds, cmd)
				}
			}
			return m, tea.Batch(cmds...)
		}

		// List mode
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.webhooks)-1 {
				m.cursor++
			}
		case "a", "A":
			m.currentMode = modeAdd
			m.focusIndex = 0
			for i := range m.inputs {
				m.inputs[i].SetValue("")
			}
			cmds = append(cmds, m.updateFocus()...)
			return m, tea.Batch(cmds...)
		case "d", "D", "delete":
			if len(m.webhooks) > 0 {
				err := m.engine.DeleteWebhook(m.webhooks[m.cursor].ID)
				if err != nil {
					m.status = "❌ Error: " + err.Error()
				} else {
					m.status = "✅ Deleted"
					m.webhooks = m.engine.GetWebhooks()
					if m.cursor >= len(m.webhooks) {
						m.cursor = len(m.webhooks) - 1
					}
					if m.cursor < 0 {
						m.cursor = 0
					}
				}
			}
		case "t", "T":
			if len(m.webhooks) > 0 {
				err := m.engine.ToggleWebhook(m.webhooks[m.cursor].ID)
				if err != nil {
					m.status = "❌ Error: " + err.Error()
				} else {
					m.status = "✅ Toggled"
					m.webhooks = m.engine.GetWebhooks()
				}
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) updateFocus() []tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return cmds
}

func (m Model) View() string {
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim
	accentColor := theme.Current.Accent

	if m.currentMode == modeAdd {
		var b strings.Builder
		b.WriteString(components.Title("ADD WEBHOOK") + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render("Name:\n"))
		b.WriteString(m.inputs[0].View() + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render("Type (discord/slack):\n"))
		b.WriteString(m.inputs[1].View() + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render("URL:\n"))
		b.WriteString(m.inputs[2].View() + "\n\n")
		
		btn := "[ Submit ]"
		if m.focusIndex == len(m.inputs) {
			btn = lipgloss.NewStyle().Foreground(theme.Current.Bg).Background(primaryColor).Bold(true).Render(btn)
		} else {
			btn = lipgloss.NewStyle().Foreground(dimColor).Render(btn)
		}
		
		b.WriteString(btn + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Error).Render(m.status) + "\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Press ESC to cancel."))
		return components.Card(b.String(), 90)
	}

	var b strings.Builder
	b.WriteString(components.Title("ALERTS & WEBHOOKS") + "\n\n")

	if len(m.webhooks) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("No webhooks configured.\n"))
	} else {
		for i, w := range m.webhooks {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
			}
			
			statusLabel := lipgloss.NewStyle().Foreground(theme.Current.Success).Render("ON")
			if !w.Enabled {
				statusLabel = lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("OFF")
			}

			nameStr := lipgloss.NewStyle().Foreground(accentColor).Width(20).Render(w.Name)
			typeStr := lipgloss.NewStyle().Foreground(theme.Current.Warning).Width(10).Render(string(w.Type))
			
			errStr := w.LastError
			if len(errStr) > 40 {
				errStr = errStr[:37] + "..."
			}

			b.WriteString(fmt.Sprintf("%s[%s] %s %s %s\n", cursor, statusLabel, nameStr, typeStr, style.Render(errStr)))
		}
	}
	
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("[A] Add   [D] Delete   [T] Toggle   [Up/Down] Navigate"))
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("Status: "+m.status))

	return components.Card(b.String(), 120)
}

func (m Model) IsInputActive() bool {
	return m.currentMode == modeAdd
}

func (m Model) Title() string {
	return "Alerts"
}

func (m Model) Icon() string {
	return "🔔"
}
