package servers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	"main/internal/config"
	"main/internal/pages"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type mode int

const (
	modeList mode = iota
	modeForm
	modeBulkAction
)

type BulkJob struct {
	ServerIdx int
	Status    string
	Output    string
	Done      bool
	Success   bool
}

type Model struct {
	servers []config.ServerConfig
	cursor  int
	status  string
	client  *sshlib.Client

	currentMode mode
	inputs      []textinput.Model
	focusIndex  int

	selected    map[int]bool
	bulkCommand string
	bulkJobs    []*BulkJob
	bulkInput   textinput.Model
}

func New() Model {
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg.Servers = []config.ServerConfig{}
	}

	bInput := textinput.New()
	bInput.Placeholder = "Command to run (e.g. apt update, systemctl restart nginx)"
	bInput.Focus()
	bInput.CharLimit = 256
	bInput.Width = 50

	m := Model{
		servers:     cfg.Servers,
		cursor:      0,
		status:      "Select server (Enter), Multi-select (Space), Bulk action (B)",
		currentMode: modeList,
		inputs:      make([]textinput.Model, 5),
		selected:    make(map[int]bool),
		bulkInput:   bInput,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		t.CharLimit = 64
		switch i {
		case 0:
			t.Placeholder = "Server Name (e.g. Production Web)"
			t.Focus()
		case 1:
			t.Placeholder = "Host / IP Address"
		case 2:
			t.Placeholder = "Port (22)"
			t.SetValue("22")
		case 3:
			t.Placeholder = "Username (root)"
			t.SetValue("root")
		case 4:
			t.Placeholder = "Password OR Path to SSH Key (~/.ssh/id_rsa)"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
		}
		m.inputs[i] = t
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.currentMode == modeForm {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.currentMode = modeList
				return m, nil

			case "tab", "shift+tab", "enter", "up", "down":
				s := msg.String()

				if s == "enter" && m.focusIndex == len(m.inputs) {
					// Save the new server
					newServer := config.ServerConfig{
						Name:     m.inputs[0].Value(),
						Host:     m.inputs[1].Value(),
						Port:     m.inputs[2].Value(),
						User:     m.inputs[3].Value(),
						Password: m.inputs[4].Value(),
					}

					// Auto-detect if the password field is actually an SSH Key path
					pwd := newServer.Password
					if strings.HasPrefix(pwd, "~") || strings.HasPrefix(pwd, "/") || strings.HasPrefix(pwd, "C:\\") {
						newServer.KeyPath = pwd
						newServer.Password = ""
					}

					m.servers = append(m.servers, newServer)
					saveConfig(m.servers)

					// Reset form
					m.currentMode = modeList
					m.cursor = len(m.servers) - 1 // Jump to new server
					m.status = "✅ Server saved successfully!"
					return m, nil
				}

				// Cycle inputs
				if s == "up" || s == "shift+tab" {
					m.focusIndex--
				} else {
					m.focusIndex++
				}

				if m.focusIndex > len(m.inputs) {
					m.focusIndex = 0
				} else if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs)
				}

				cmds := make([]tea.Cmd, len(m.inputs))
				for i := 0; i <= len(m.inputs)-1; i++ {
					if i == m.focusIndex {
						cmds[i] = m.inputs[i].Focus()
						continue
					}
					m.inputs[i].Blur()
				}
				return m, tea.Batch(cmds...)
			}

			// Handle character input
			cmd := m.updateInputs(msg)
			return m, cmd
		}

		if m.currentMode == modeBulkAction {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.currentMode = modeList
				m.bulkJobs = nil
				m.bulkInput.SetValue("")
				return m, nil
			case "enter":
				if m.bulkJobs != nil {
					return m, nil
				}
				return m.startBulkExecution()
			default:
				if m.bulkJobs == nil {
					var cmd tea.Cmd
					m.bulkInput, cmd = m.bulkInput.Update(msg)
					return m, cmd
				}
				return m, nil
			}
		}

		// List Mode
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.servers) { // Allow cursor to reach the 'Add New' button
				m.cursor++
			}
		case " ":
			if m.cursor < len(m.servers) {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "b", "B":
			hasSelected := false
			for _, v := range m.selected {
				if v { hasSelected = true }
			}
			if hasSelected {
				m.currentMode = modeBulkAction
				m.bulkInput.Focus()
				m.bulkJobs = nil
			}
		case "enter":
			if m.cursor == len(m.servers) {
				m.currentMode = modeForm
				return m, nil
			}
			return m.instantConnect()
		}

	case bulkJobResult:
		var cmd tea.Cmd
		for _, j := range m.bulkJobs {
			if j.ServerIdx == msg.serverIdx {
				j.Done = true
				j.Success = msg.success
				j.Output = msg.output
				
				sName := m.servers[j.ServerIdx].Name
				
				if j.Success {
					j.Status = "Completed"
					cmd = func() tea.Msg {
						return pages.LogActivityMsg{Message: fmt.Sprintf("Bulk action on %s completed successfully", sName)}
					}
				} else {
					j.Status = "Failed"
					cmd = func() tea.Msg {
						return pages.LogActivityMsg{Message: fmt.Sprintf("Bulk action on %s failed", sName)}
					}
				}
			}
		}
		return m, cmd
	case error:
		m.status = msg.Error()
		return m, nil
	case sshlib.ConnectedMsg:
		m.status = "✅ Connected! Deploying Agent..."
		m.client = msg.Client
		return m, nil
	}
	return m, nil
}

type bulkJobResult struct {
	serverIdx int
	success   bool
	output    string
}

func (m *Model) startBulkExecution() (tea.Model, tea.Cmd) {
	cmdStr := m.bulkInput.Value()
	if cmdStr == "" {
		return *m, nil
	}

	m.bulkCommand = cmdStr
	m.bulkJobs = []*BulkJob{}
	var cmds []tea.Cmd

	for i, s := range m.servers {
		if m.selected[i] {
			job := &BulkJob{
				ServerIdx: i,
				Status:    "Running...",
			}
			m.bulkJobs = append(m.bulkJobs, job)

			serverCfg := s
			idx := i
			cmds = append(cmds, func() tea.Msg {
				client, err := sshlib.Connect(serverCfg.Host, serverCfg.Port, serverCfg.User, serverCfg.Password, serverCfg.KeyPath)
				if err != nil {
					return bulkJobResult{serverIdx: idx, success: false, output: err.Error()}
				}
				defer client.Close()
				out, err := client.Run(cmdStr)
				if err != nil {
					return bulkJobResult{serverIdx: idx, success: false, output: err.Error() + "\n" + out}
				}
				return bulkJobResult{serverIdx: idx, success: true, output: out}
			})
		}
	}
	return *m, tea.Batch(cmds...)
}

func (m Model) IsInputActive() bool {
	return m.currentMode == modeForm
}

func (m *Model) instantConnect() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.servers) {
		return *m, nil
	}
	s := m.servers[m.cursor]
	m.status = fmt.Sprintf("Connecting to %s...", s.Name)

	return *m, func() tea.Msg {
		client, err := sshlib.Connect(s.Host, s.Port, s.User, s.Password, s.KeyPath)
		if err != nil {
			return fmt.Errorf("❌ Connection failed: %v", err)
		}
		return sshlib.ConnectedMsg{Client: client, Host: s.Host, Port: s.Port, User: s.User}
	}
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Model) View() string {
	accentColor := theme.Current.Accent
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim

	if m.currentMode == modeBulkAction {
		var b strings.Builder
		b.WriteString(components.Title("FLEET-WIDE BULK ACTION") + "\n\n")

		if m.bulkJobs == nil {
			// Input phase
			b.WriteString(m.bulkInput.View() + "\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Press Enter to execute across all selected servers. ESC to cancel."))
			return components.Card(b.String(), 90)
		}

		// Execution phase
		successCount := 0
		failCount := 0
		totalDone := 0
		for _, j := range m.bulkJobs {
			if j.Done {
				totalDone++
				if j.Success {
					successCount++
				} else {
					failCount++
				}
			}
		}

		header := fmt.Sprintf("Summary: %d/%d completed. %d succeeded, %d failed.\n\n", totalDone, len(m.bulkJobs), successCount, failCount)
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(header))

		for _, j := range m.bulkJobs {
			sName := m.servers[j.ServerIdx].Name
			statusStr := ""
			if !j.Done {
				statusStr = lipgloss.NewStyle().Foreground(theme.Current.Warning).Render("⏳ Running...")
			} else if j.Success {
				statusStr = lipgloss.NewStyle().Foreground(theme.Current.Success).Render("✓ Completed")
			} else {
				statusStr = lipgloss.NewStyle().Foreground(theme.Current.Error).Render("✗ Failed")
			}
			b.WriteString(fmt.Sprintf("%-20s %s\n", sName, statusStr))
			if j.Done && !j.Success {
				errLines := strings.Split(j.Output, "\n")
				for k, l := range errLines {
					if l != "" && k < 5 { // limit error output lines
						b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("  │ " + l) + "\n")
					}
				}
			}
		}
		
		if totalDone == len(m.bulkJobs) {
			b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("\nPress ESC to return."))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("\nExecuting..."))
		}
		
		return components.Card(b.String(), 90)
	}

	if m.currentMode == modeForm {
		var b strings.Builder
		b.WriteString(components.Title("ADD NEW SERVER") + "\n\n")

		for i := range m.inputs {
			b.WriteString(m.inputs[i].View())
			if i < len(m.inputs)-1 {
				b.WriteRune('\n')
			}
		}

		button := "[ Submit ]"
		if m.focusIndex == len(m.inputs) {
			button = lipgloss.NewStyle().Foreground(theme.Current.Bg).Background(primaryColor).Bold(true).Render(button)
		} else {
			button = lipgloss.NewStyle().Foreground(dimColor).Render(button)
		}
		
		b.WriteString("\n\n" + button + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Press ESC to cancel."))
		return components.Card(b.String(), 60)
	}

	var items string
	items += components.Title("REGISTERED SERVERS") + "\n\n"

	for i, s := range m.servers {
		cursor := " [ ] "
		if m.selected[i] { cursor = " [x] " }
		
		style := lipgloss.NewStyle().Foreground(theme.Current.Text)
		
		if m.cursor == i {
			if m.selected[i] { cursor = "▶[x] " } else { cursor = "▶[ ] " }
			style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
		}

		icon := "🟢"
		lowerName := strings.ToLower(s.Name)
		if strings.Contains(lowerName, "dev") {
			icon = "🔴"
		} else if strings.Contains(lowerName, "backup") {
			icon = "🟡"
		} else if m.client == nil && m.cursor != i {
			icon = "⚪" // not connected
		}

		items += fmt.Sprintf("%s %s %s %s\n", cursor, icon, style.Render(s.Name), lipgloss.NewStyle().Foreground(dimColor).Render(s.User+"@"+s.Host))
	}

	// Add new button
	cursor := "  "
	btnStyle := lipgloss.NewStyle().Foreground(dimColor)
	if m.cursor == len(m.servers) {
		cursor = "▶ "
		btnStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	}
	items += fmt.Sprintf("\n%s %s\n", cursor, btnStyle.Render("[+] Add New Server"))
	
	items += "\n" + lipgloss.NewStyle().Foreground(accentColor).Render(m.status)

	return components.Card(items, 60)
}

func (m Model) Title() string { return "Servers" }
func (m Model) Icon() string { return "🖥️" }

func saveConfig(servers []config.ServerConfig) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".vortex", "config.json")
	data, _ := json.MarshalIndent(config.Config{Servers: servers}, "", "  ")
	os.WriteFile(configPath, data, 0644)
}
