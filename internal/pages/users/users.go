package users

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	userengine "main/internal/engine/users"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type viewState int

const (
	stateLoading viewState = iota
	stateListUsers
	stateUserDetails
	stateAddKey
	stateConfirmRevoke
)

type Model struct {
	engine       *userengine.Engine
	status       string
	state        viewState
	loading      bool

	users        []userengine.User
	userCursor   int
	
	keyCursor    int

	keyInput     textinput.Model
}

func New() Model {
	ki := textinput.New()
	ki.Placeholder = "ssh-rsa AAAA..."
	ki.Width = 60

	return Model{
		status:     "Connecting...",
		state:      stateLoading,
		keyInput:   ki,
	}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

type usersLoadedMsg []userengine.User
type operationCompleteMsg struct{ err error }

func loadUsers(engine *userengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return usersLoadedMsg{}
		}
		users, err := engine.GetUsers()
		if err != nil {
			return usersLoadedMsg{}
		}
		return usersLoadedMsg(users)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case usersLoadedMsg:
		m.users = []userengine.User(msg)
		if m.userCursor >= len(m.users) {
			m.userCursor = len(m.users) - 1
			if m.userCursor < 0 {
				m.userCursor = 0
			}
		}
		m.state = stateListUsers
		m.loading = false
		m.status = "Idle"
		return m, nil

	case operationCompleteMsg:
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
		} else {
			m.status = "Success."
		}
		m.state = stateLoading
		m.loading = true
		return m, loadUsers(m.engine)

	case tea.KeyMsg:
		if m.state == stateLoading {
			return m, nil
		}

		if m.state == stateListUsers {
			switch msg.String() {
			case "up", "k":
				if m.userCursor > 0 { m.userCursor-- }
			case "down", "j":
				if m.userCursor < len(m.users)-1 { m.userCursor++ }
			case "enter":
				if len(m.users) > 0 {
					m.state = stateUserDetails
					m.keyCursor = 0
				}
			case "r", "R":
				m.state = stateLoading
				m.loading = true
				m.status = "Refreshing users..."
				return m, loadUsers(m.engine)
			}
		} else if m.state == stateUserDetails {
			switch msg.String() {
			case "esc":
				m.state = stateListUsers
			case "up", "k":
				u := m.users[m.userCursor]
				if m.keyCursor > 0 { m.keyCursor-- } else { m.keyCursor = len(u.Keys) - 1 }
			case "down", "j":
				u := m.users[m.userCursor]
				if m.keyCursor < len(u.Keys)-1 { m.keyCursor++ } else { m.keyCursor = 0 }
			case "a", "A":
				m.state = stateAddKey
				m.keyInput.SetValue("")
				m.keyInput.Focus()
			case "d", "D", "delete":
				u := m.users[m.userCursor]
				if len(u.Keys) > 0 {
					m.state = stateConfirmRevoke
				} else {
					m.status = "No keys to revoke."
				}
			}
		} else if m.state == stateAddKey {
			switch msg.String() {
			case "esc":
				m.state = stateUserDetails
			case "enter":
				u := m.users[m.userCursor]
				keyStr := m.keyInput.Value()
				if strings.TrimSpace(keyStr) == "" {
					m.status = "Key cannot be empty."
					return m, nil
				}
				m.state = stateLoading
				m.loading = true
				m.status = "Adding key..."
				return m, func() tea.Msg {
					err := m.engine.AddKey(u.Username, u.HomeDir, keyStr)
					return operationCompleteMsg{err: err}
				}
			}
			m.keyInput, cmd = m.keyInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.state == stateConfirmRevoke {
			switch msg.String() {
			case "esc", "n", "N":
				m.state = stateUserDetails
				m.status = "Revocation cancelled."
			case "y", "Y", "enter":
				u := m.users[m.userCursor]
				idx := m.keyCursor
				m.state = stateLoading
				m.loading = true
				m.status = "Revoking key..."
				return m, func() tea.Msg {
					err := m.engine.RevokeKey(u.Username, u.HomeDir, idx, u.Keys)
					return operationCompleteMsg{err: err}
				}
			}
		}

	case sshlib.ConnectedMsg:
		m.engine = userengine.NewEngine(msg.Client)
		m.status = "Connected. Fetching users..."
		m.loading = true
		return m, loadUsers(m.engine)

	case agent.Payload:
		if m.engine != nil && m.state == stateLoading && !m.loading {
			m.loading = true
			return m, loadUsers(m.engine)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.state == stateLoading {
		return components.Card(lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.status), 60)
	}

	accentColor := theme.Current.Accent
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim
	errorColor := theme.Current.Error

	var b strings.Builder
	b.WriteString(components.Title("USER & SSH KEY MANAGER") + "\n\n")

	if m.state == stateListUsers {
		if len(m.users) == 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("No users found.\n"))
		} else {
			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(dimColor)
			b.WriteString(headerStyle.Render(fmt.Sprintf("  %-15s %-8s %-25s %s\n", "USERNAME", "UID", "HOME", "KEYS/RISK")))
			for i, u := range m.users {
				cursor := "  "
				style := lipgloss.NewStyle().Foreground(theme.Current.Text)
				if m.userCursor == i {
					cursor = "▶ "
					style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
				}

				keyInfo := fmt.Sprintf("%d keys", len(u.Keys))
				if u.Risky {
					keyInfo = lipgloss.NewStyle().Foreground(errorColor).Bold(true).Render(keyInfo + " (RISKY)")
				} else {
					keyInfo = lipgloss.NewStyle().Foreground(accentColor).Render(keyInfo)
				}

				b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(fmt.Sprintf("%-15s %-8s %-25s %s", u.Username, u.UID, u.HomeDir, keyInfo))))
			}
		}
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("[Enter] Manage User   [R] Refresh   [Up/Down] Navigate"))

	} else if m.state == stateUserDetails || m.state == stateConfirmRevoke {
		u := m.users[m.userCursor]
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("User: " + u.Username))
		if u.Risky {
			b.WriteString(" " + lipgloss.NewStyle().Foreground(theme.Current.Bg).Background(errorColor).Bold(true).Render(" RISKY "))
			b.WriteString(lipgloss.NewStyle().Foreground(errorColor).Render(" (Missing SSH keys or root login enabled)"))
		}
		b.WriteString("\n\n")

		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Authorized Keys:") + "\n")
		if len(u.Keys) == 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("  No SSH keys configured for this user.\n"))
		} else {
			for i, k := range u.Keys {
				cursor := "  "
				style := lipgloss.NewStyle().Foreground(theme.Current.Text)
				if m.keyCursor == i {
					cursor = "▶ "
					style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
				}

				preview := k.Raw
				if len(preview) > 60 {
					preview = preview[:30] + "..." + preview[len(preview)-27:]
				}
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(fmt.Sprintf("[%d] %s (%s)", i, preview, k.Type))))
			}
		}

		if m.state == stateConfirmRevoke {
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Bg).Background(errorColor).Bold(true).Render(fmt.Sprintf(" REVOKE KEY [%d]? (Y/N) ", m.keyCursor)))
			b.WriteString("\n")
		} else {
			b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("[A] Add Key   [D] Revoke Key   [Esc] Back   [Up/Down] Navigate"))
		}
	} else if m.state == stateAddKey {
		u := m.users[m.userCursor]
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("Add SSH Key for: " + u.Username))
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Paste the public SSH key below:") + "\n")
		b.WriteString(m.keyInput.View() + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("[Enter] Submit   [Esc] Cancel"))
	}

	b.WriteString("\n\n" + lipgloss.NewStyle().Foreground(dimColor).Render("Status: "+m.status))
	return components.Card(b.String(), 90)
}

func (m Model) IsInputActive() bool {
	return m.state == stateAddKey
}

func (m Model) Title() string { return "Users & Keys" }
func (m Model) Icon() string  { return "👥" }
