package servers

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sshlib "main/internal/ssh"
)

// ServerConfig holds the connection details for a remote VPS
type ServerConfig struct {
	Name     string
	Host     string
	Port     string
	User     string
	Password string
	KeyPath  string
}

// Model holds the state for the Server Manager page
type Model struct {
	servers []ServerConfig
	cursor  int
	status  string
	client  *sshlib.Client
}

func New() Model {
	return Model{
		servers: []ServerConfig{
			{Name: "Localhost (Agent Mode)", Host: "127.0.0.1", Port: "22", User: "root", Password: "password"},
			{Name: "Production Web", Host: "192.168.1.100", Port: "22", User: "admin", KeyPath: "~/.ssh/id_rsa"},
			{Name: "Database Cluster", Host: "10.0.0.5", Port: "22", User: "root", KeyPath: "~/.ssh/id_ed25519"},
		},
		cursor: 0,
		status: "Select a server to connect",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.servers)-1 {
				m.cursor++
			}
		case "enter":
			s := m.servers[m.cursor]
			m.status = fmt.Sprintf("Connecting to %s...", s.Name)
			
			// Establish connection asynchronously to avoid UI freeze
			return m, func() tea.Msg {
				client, err := sshlib.Connect(s.Host, s.Port, s.User, s.Password, s.KeyPath)
				if err != nil {
					return fmt.Errorf("❌ Connection failed: %v", err)
				}
				return sshlib.ConnectedMsg{Client: client, Host: s.Host, User: s.User}
			}
		}
	case error:
		// Handle the async connection failure
		m.status = msg.Error()
		return m, nil
	case sshlib.ConnectedMsg:
		m.status = "✅ Connected! Deploying Agent..."
		m.client = msg.Client
		return m, nil
	}
	return m, nil
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
