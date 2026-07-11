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

	"main/internal/config"
	sshlib "main/internal/ssh"
)

type mode int

const (
	modeList mode = iota
	modeForm
)

type Model struct {
	servers []config.ServerConfig
	cursor  int
	status  string
	client  *sshlib.Client

	currentMode mode
	inputs      []textinput.Model
	focusIndex  int
}

func New() Model {
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg.Servers = []config.ServerConfig{}
	}

	m := Model{
		servers:     cfg.Servers,
		cursor:      0,
		status:      "Select a server (Up/Down arrows, Enter to connect)",
		currentMode: modeList,
		inputs:      make([]textinput.Model, 5),
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
		case "enter":
			if m.cursor == len(m.servers) {
				m.currentMode = modeForm
				return m, nil
			}

			s := m.servers[m.cursor]
			m.status = fmt.Sprintf("Connecting to %s...", s.Name)

			// Establish connection asynchronously
			return m, func() tea.Msg {
				client, err := sshlib.Connect(s.Host, s.Port, s.User, s.Password, s.KeyPath)
				if err != nil {
					return fmt.Errorf("❌ Connection failed: %v", err)
				}
				return sshlib.ConnectedMsg{Client: client, Host: s.Host, User: s.User}
			}
		}

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

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m Model) View() string {
	accentColor := lipgloss.Color("205")
	primaryColor := lipgloss.Color("86")
	dimColor := lipgloss.Color("240")

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dimColor).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		MarginBottom(1)

	if m.currentMode == modeForm {
		var form strings.Builder
		form.WriteString(titleCard.Render("➕ ADD NEW SERVER") + "\n\n")

		labels := []string{"Name:", "Host:", "Port:", "User:", "Auth:"}
		for i := range m.inputs {
			form.WriteString(fmt.Sprintf("%-10s %s\n", labels[i], m.inputs[i].View()))
		}

		button := lipgloss.NewStyle().Foreground(dimColor).Render("[ Submit ]")
		if m.focusIndex == len(m.inputs) {
			button = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("[ Submit ]")
		}
		form.WriteString("\n" + button + "\n\n(Press Esc to cancel)")

		return card.Render(form.String())
	}

	// List Mode View
	var items string
	for i, s := range m.servers {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		if m.cursor == i {
			cursor = "▶ "
			style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
		}
		items += fmt.Sprintf("%s %s\n", cursor, style.Render(s.Name+" ("+s.User+"@"+s.Host+")"))
	}

	// Add New Server Button
	addCursor := "  "
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	if m.cursor == len(m.servers) {
		addCursor = "▶ "
		addStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	}
	items += fmt.Sprintf("%s %s\n", addCursor, addStyle.Render("[+] Add New Server"))

	statusBlock := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(m.status)

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("SERVER CONNECTIONS"),
			items,
			"",
			statusBlock,
		),
	)
}

func (m Model) Title() string { return "Servers" }
func (m Model) Icon() string { return "🖥️" }

func saveConfig(servers []config.ServerConfig) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".vortex", "config.json")
	data, _ := json.MarshalIndent(config.Config{Servers: servers}, "", "  ")
	os.WriteFile(configPath, data, 0644)
}
