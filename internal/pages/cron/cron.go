package cron

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/textinput"

	"main/internal/agent"
	"main/internal/components"
	cronengine "main/internal/engine/cron"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type mode int

const (
	modeList mode = iota
	modeAdd
)

type Model struct {
	jobs    []cronengine.CronJob
	engine  *cronengine.Engine
	cursor  int
	loading bool
	status  string

	currentMode mode
	inputs      []textinput.Model
	focusIndex  int
}

func New() Model {
	inputs := make([]textinput.Model, 2)
	
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "* * * * *"
	inputs[0].Focus()
	
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "/usr/bin/certbot renew"
	
	return Model{
		jobs:        []cronengine.CronJob{},
		loading:     false,
		status:      "Connecting to Cron Engine...",
		currentMode: modeList,
		inputs:      inputs,
	}
}

func (m Model) Init() tea.Cmd { return nil }

type jobsLoadedMsg []cronengine.CronJob
type operationCompleteMsg struct{ err error }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.currentMode == modeAdd {
			switch msg.String() {
			case "esc":
				m.currentMode = modeList
				return m, nil
			case "enter":
				if m.focusIndex == len(m.inputs) {
					// Submit
					schedule := m.inputs[0].Value()
					cmdStr := m.inputs[1].Value()
					
					// Basic syntax check
					if len(strings.Fields(schedule)) != 5 {
						m.status = "❌ Invalid cron syntax. Expected 5 fields."
						return m, nil
					}
					
					return m, func() tea.Msg {
						err := m.engine.AddJob(schedule, cmdStr)
						return operationCompleteMsg{err: err}
					}
				}
			case "up", "shift+tab":
				m.focusIndex--
				if m.focusIndex < 0 { m.focusIndex = len(m.inputs) }
				cmds = append(cmds, m.updateFocus()...)
				return m, tea.Batch(cmds...)
			case "down", "tab":
				m.focusIndex++
				if m.focusIndex > len(m.inputs) { m.focusIndex = 0 }
				cmds = append(cmds, m.updateFocus()...)
				return m, tea.Batch(cmds...)
			}
			
			// Handle text input
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
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.jobs)-1 { m.cursor++ }
		case "a", "A":
			m.currentMode = modeAdd
			m.focusIndex = 0
			m.inputs[0].SetValue("")
			m.inputs[1].SetValue("")
			cmds = append(cmds, m.updateFocus()...)
			return m, tea.Batch(cmds...)
		case "d", "D", "delete":
			if len(m.jobs) > 0 {
				job := m.jobs[m.cursor]
				if job.User != "user" { // Assuming "user" is the only writable crontab safely here
					m.status = "⚠️ Cannot delete system crontabs directly."
					return m, nil
				}
				return m, func() tea.Msg {
					err := m.engine.DeleteJobRaw(job.Raw)
					return operationCompleteMsg{err: err}
				}
			}
		}

	case sshlib.ConnectedMsg:
		m.engine = cronengine.NewEngine(msg.Client)
		m.status = "Connected. Fetching jobs..."
		m.loading = true
		return m, func() tea.Msg {
			jobs, err := m.engine.ListJobs()
			if err != nil {
				return jobsLoadedMsg{}
			}
			return jobsLoadedMsg(jobs)
		}

	case agent.Payload:
		if m.engine != nil && !m.loading {
			m.loading = true
			return m, func() tea.Msg {
				jobs, _ := m.engine.ListJobs()
				return jobsLoadedMsg(jobs)
			}
		}

	case jobsLoadedMsg:
		m.jobs = []cronengine.CronJob(msg)
		m.loading = false
		m.status = "Idle"
		if m.cursor >= len(m.jobs) {
			m.cursor = len(m.jobs) - 1
			if m.cursor < 0 { m.cursor = 0 }
		}

	case operationCompleteMsg:
		if msg.err != nil {
			m.status = "❌ Error: " + msg.err.Error()
		} else {
			m.status = "✅ Success!"
			m.currentMode = modeList
			// Refresh jobs
			m.loading = true
			return m, func() tea.Msg {
				jobs, _ := m.engine.ListJobs()
				return jobsLoadedMsg(jobs)
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

// humanizeCron does a very basic translation of common cron syntax
func humanizeCron(schedule string) string {
	if schedule == "* * * * *" { return "Every minute" }
	if schedule == "0 * * * *" { return "Every hour" }
	if schedule == "0 0 * * *" { return "Every day at midnight" }
	if schedule == "0 0 * * 0" { return "Every Sunday at midnight" }
	return schedule
}

func (m Model) View() string {
	accentColor := theme.Current.Accent
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim

	if m.currentMode == modeAdd {
		var b strings.Builder
		b.WriteString(components.Title("ADD CRON JOB") + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render("Schedule (e.g. 0 5 * * *):\n"))
		b.WriteString(m.inputs[0].View() + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render("Command:\n"))
		b.WriteString(m.inputs[1].View() + "\n\n")
		
		btn := "[ Submit ]"
		if m.focusIndex == 2 {
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
	b.WriteString(components.Title("SCHEDULED TASKS (CRON)") + "\n\n")
	
	if len(m.jobs) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("No cron jobs found.\n"))
	} else {
		for i, j := range m.jobs {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
			}
			
			human := humanizeCron(j.Schedule)
			schedStr := lipgloss.NewStyle().Foreground(accentColor).Width(25).Render(j.Schedule + " (" + human + ")")
			userStr := lipgloss.NewStyle().Foreground(theme.Current.Warning).Width(10).Render(j.User)
			cmdStr := lipgloss.NewStyle().Foreground(dimColor).Render(j.Command)
			
			if len(cmdStr) > 50 { cmdStr = cmdStr[:47] + "..." }
			
			b.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, userStr, schedStr, style.Render(cmdStr)))
		}
	}
	
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("[A] Add Job   [D] Delete Job   [Up/Down] Navigate"))
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("Status: "+m.status))

	return components.Card(b.String(), 120)
}

func (m Model) IsInputActive() bool {
	return m.currentMode == modeAdd
}

func (m Model) Title() string { return "Cron" }
func (m Model) Icon() string { return "🕒" }
