package certs

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	certsengine "main/internal/engine/certs"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	certs   []certsengine.CertInfo
	engine  *certsengine.Engine
	cursor  int
	loading bool
	status  string

	isRenewing  bool
	renewOutput string
}

func New() Model {
	return Model{
		certs:   []certsengine.CertInfo{},
		loading: false,
		status:  "Connecting to Certs Engine...",
	}
}

func (m Model) Init() tea.Cmd { return nil }

type certsLoadedMsg []certsengine.CertInfo
type certsErrorMsg string
type renewCompleteMsg struct{ err error; output string }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isRenewing {
			if msg.String() == "esc" {
				m.isRenewing = false
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(m.certs)-1 { m.cursor++ }
		case "r", "R":
			if len(m.certs) > 0 {
				cert := m.certs[m.cursor]
				m.isRenewing = true
				m.renewOutput = "Running certbot renew for " + cert.Name + "...\n"
				m.status = "Renewing..."
				return m, func() tea.Msg {
					out, err := m.engine.RenewCertificate(cert.Name)
					return renewCompleteMsg{err: err, output: out}
				}
			}
		}

	case sshlib.ConnectedMsg:
		m.engine = certsengine.NewEngine(msg.Client)
		m.status = "Connected. Scanning for SSL Certificates..."
		m.loading = true
		return m, func() tea.Msg {
			certs, err := m.engine.ListCertificates()
			if err != nil {
				return certsErrorMsg(err.Error())
			}
			return certsLoadedMsg(certs)
		}

	case agent.Payload:
		if m.engine != nil && !m.loading && !m.isRenewing {
			m.loading = true
			return m, func() tea.Msg {
				certs, err := m.engine.ListCertificates()
				if err != nil {
					return certsErrorMsg(err.Error())
				}
				return certsLoadedMsg(certs)
			}
		}

	case certsLoadedMsg:
		m.certs = []certsengine.CertInfo(msg)
		m.loading = false
		m.status = "Idle"
		if m.cursor >= len(m.certs) {
			m.cursor = len(m.certs) - 1
			if m.cursor < 0 { m.cursor = 0 }
		}
		
		// Proactive warning if any cert is expiring
		if len(m.certs) > 0 && m.certs[0].DaysLeft < 14 {
			// This could trigger a system notification if wired up
			m.status = fmt.Sprintf("⚠️ WARNING: %s expires in %d days!", m.certs[0].Name, m.certs[0].DaysLeft)
		}

	case certsErrorMsg:
		m.loading = false
		m.status = string(msg)

	case renewCompleteMsg:
		if msg.err != nil {
			m.renewOutput += "\n\n❌ Error:\n" + msg.err.Error() + "\n" + msg.output
			m.status = "Renewal Failed"
		} else {
			m.renewOutput += "\n\n✅ Renewal Successful!\n" + msg.output
			m.status = "Renewal Successful"
			
			// Refresh list
			return m, func() tea.Msg {
				certs, _ := m.engine.ListCertificates()
				return certsLoadedMsg(certs)
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim

	if m.isRenewing {
		var b strings.Builder
		b.WriteString(components.Title("CERTIFICATE RENEWAL") + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Text).Render(m.renewOutput) + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Press ESC to return."))
		return components.Card(b.String(), 90)
	}

	var b strings.Builder
	b.WriteString(components.Title("SSL CERTIFICATE MANAGER") + "\n\n")
	
	if m.loading && len(m.certs) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("Scanning certificates...\n"))
	} else if len(m.certs) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Warning).Render("No certificates found or certbot is not installed.\n"))
	} else {
		for i, c := range m.certs {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
			}
			
			// Semantic color by days-until-expiry
			var daysStyle lipgloss.Style
			if c.DaysLeft > 30 {
				daysStyle = lipgloss.NewStyle().Foreground(theme.Current.Success)
			} else if c.DaysLeft >= 7 {
				daysStyle = lipgloss.NewStyle().Foreground(theme.Current.Warning)
			} else {
				daysStyle = lipgloss.NewStyle().Foreground(theme.Current.Error).Bold(true)
			}
			
			daysStr := daysStyle.Width(15).Render(fmt.Sprintf("%d days left", c.DaysLeft))
			nameStr := style.Width(35).Render(c.Name)
			domainsStr := lipgloss.NewStyle().Foreground(dimColor).Render(strings.Join(c.Domains, ", "))
			if len(domainsStr) > 40 { domainsStr = domainsStr[:37] + "..." }
			
			b.WriteString(fmt.Sprintf("%s%s %s %s\n", cursor, nameStr, daysStr, domainsStr))
		}
	}
	
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("[R] Renew Certificate   [Up/Down] Navigate"))
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("Status: "+m.status))

	return components.Card(b.String(), 120)
}

func (m Model) IsInputActive() bool {
	return false
}

func (m Model) Title() string { return "Certs" }
func (m Model) Icon() string { return "🔒" }
