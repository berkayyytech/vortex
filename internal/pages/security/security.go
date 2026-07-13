package security

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	secengine "main/internal/engine/security"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type viewState int

const (
	stateLoading viewState = iota
	stateSummary
	stateFirewall
	statePorts
	stateFail2Ban
	stateAddRule
	stateDeleteRule
	stateBanIP
	stateUnbanIP
)

type Model struct {
	report *secengine.FullAuditReport
	engine *secengine.Engine
	status string
	state  viewState

	// Form inputs
	portInput   textinput.Model
	protoInput  textinput.Model
	actionInput textinput.Model
	idInput     textinput.Model
	ipInput     textinput.Model
	jailInput   textinput.Model

	formStep int
}

func New() Model {
	port := textinput.New()
	port.Placeholder = "e.g. 80, 443, 8080"
	port.Focus()

	proto := textinput.New()
	proto.Placeholder = "tcp, udp, or any"

	action := textinput.New()
	action.Placeholder = "allow or deny"

	idInp := textinput.New()
	idInp.Placeholder = "Rule ID"

	ipInp := textinput.New()
	ipInp.Placeholder = "IP Address"

	jailInp := textinput.New()
	jailInp.Placeholder = "Jail Name (e.g. sshd)"

	return Model{
		status:      "Connecting...",
		state:       stateLoading,
		portInput:   port,
		protoInput:  proto,
		actionInput: action,
		idInput:     idInp,
		ipInput:     ipInp,
		jailInput:   jailInp,
	}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) IsInputActive() bool {
	return m.state == stateAddRule || m.state == stateDeleteRule || m.state == stateBanIP || m.state == stateUnbanIP
}

type auditReportMsg *secengine.FullAuditReport

func runFullAudit(engine *secengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		report, err := engine.RunFullAudit()
		if err != nil {
			return nil
		}
		return auditReportMsg(report)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case auditReportMsg:
		m.report = msg
		m.status = "Audit complete."
		m.state = stateSummary
		return m, nil

	case tea.KeyMsg:
		if m.state == stateLoading {
			return m, nil
		}

		if m.state == stateSummary || m.state == stateFirewall || m.state == statePorts || m.state == stateFail2Ban {
			switch msg.String() {
			case "r", "R":
				m.status = "Running deep security audit..."
				m.state = stateLoading
				return m, runFullAudit(m.engine)
			case "1":
				m.state = stateSummary
			case "2":
				m.state = stateFirewall
			case "3":
				m.state = statePorts
			case "4":
				m.state = stateFail2Ban
			}
		}

		if m.state == stateFirewall {
			switch msg.String() {
			case "a", "A":
				m.state = stateAddRule
				m.formStep = 0
				m.portInput.SetValue("")
				m.protoInput.SetValue("")
				m.actionInput.SetValue("")
				m.portInput.Focus()
				m.protoInput.Blur()
				m.actionInput.Blur()
			case "d", "D":
				m.state = stateDeleteRule
				m.idInput.SetValue("")
				m.idInput.Focus()
			}
		} else if m.state == stateFail2Ban {
			switch msg.String() {
			case "b", "B":
				m.state = stateBanIP
				m.formStep = 0
				m.ipInput.SetValue("")
				m.jailInput.SetValue("")
				m.ipInput.Focus()
				m.jailInput.Blur()
			case "u", "U":
				m.state = stateUnbanIP
				m.formStep = 0
				m.ipInput.SetValue("")
				m.jailInput.SetValue("")
				m.ipInput.Focus()
				m.jailInput.Blur()
			}
		} else if m.state == stateAddRule {
			switch msg.String() {
			case "esc":
				m.state = stateFirewall
			case "enter":
				if m.formStep == 0 {
					m.formStep++
					m.portInput.Blur()
					m.protoInput.Focus()
				} else if m.formStep == 1 {
					m.formStep++
					m.protoInput.Blur()
					m.actionInput.Focus()
				} else if m.formStep == 2 {
					// submit
					err := m.engine.AddFirewallRule(m.portInput.Value(), m.protoInput.Value(), m.actionInput.Value())
					if err != nil {
						m.status = "Error adding rule: " + err.Error()
					} else {
						m.status = "Rule added successfully."
					}
					m.state = stateLoading
					return m, runFullAudit(m.engine)
				}
			}
			m.portInput, cmd = m.portInput.Update(msg)
			cmds = append(cmds, cmd)
			m.protoInput, cmd = m.protoInput.Update(msg)
			cmds = append(cmds, cmd)
			m.actionInput, cmd = m.actionInput.Update(msg)
			cmds = append(cmds, cmd)

		} else if m.state == stateDeleteRule {
			switch msg.String() {
			case "esc":
				m.state = stateFirewall
			case "enter":
				err := m.engine.RemoveFirewallRule(m.idInput.Value())
				if err != nil {
					m.status = "Error deleting rule: " + err.Error()
				} else {
					m.status = "Rule deleted successfully."
				}
				m.state = stateLoading
				return m, runFullAudit(m.engine)
			}
			m.idInput, cmd = m.idInput.Update(msg)
			cmds = append(cmds, cmd)

		} else if m.state == stateBanIP || m.state == stateUnbanIP {
			switch msg.String() {
			case "esc":
				m.state = stateFail2Ban
			case "enter":
				if m.formStep == 0 {
					m.formStep++
					m.ipInput.Blur()
					m.jailInput.Focus()
				} else if m.formStep == 1 {
					var err error
					if m.state == stateBanIP {
						err = m.engine.BanIP(m.jailInput.Value(), m.ipInput.Value())
					} else {
						err = m.engine.UnbanIP(m.jailInput.Value(), m.ipInput.Value())
					}
					if err != nil {
						m.status = "Error: " + err.Error()
					} else {
						m.status = "Success."
					}
					m.state = stateLoading
					return m, runFullAudit(m.engine)
				}
			}
			m.ipInput, cmd = m.ipInput.Update(msg)
			cmds = append(cmds, cmd)
			m.jailInput, cmd = m.jailInput.Update(msg)
			cmds = append(cmds, cmd)
		}

	case sshlib.ConnectedMsg:
		m.engine = secengine.NewEngine(msg.Client)
		m.status = "Running deep security audit..."
		return m, runFullAudit(m.engine)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.state == stateLoading {
		return components.Card(lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.status), 60)
	}

	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	label := lipgloss.NewStyle().Foreground(theme.Current.Primary).Width(25).Bold(true)
	text := lipgloss.NewStyle().Foreground(theme.Current.Text)
	dim := lipgloss.NewStyle().Foreground(theme.Current.Dim)

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		components.Title("SECURITY CENTER"),
		"  ",
		dim.Render("[1] Summary   [2] Firewall   [3] Open Ports   [4] Fail2Ban"),
	)

	var content string

	switch m.state {
	case stateSummary:
		var ufwStr string
		if m.report.BasicReport.UFWStatus == "Active" {
			ufwStr = green.Render("Secure (Active)")
		} else {
			ufwStr = red.Render("VULNERABLE (Inactive/Missing)")
		}

		var rootStr string
		if m.report.BasicReport.RootLoginEnabled {
			rootStr = red.Render("VULNERABLE (Enabled)")
		} else {
			rootStr = green.Render("Secure (Disabled)")
		}

		var passStr string
		if m.report.BasicReport.PasswordAuthEnabled {
			passStr = red.Render("VULNERABLE (Passwords Allowed)")
		} else {
			passStr = green.Render("Secure (Key Only)")
		}

		items := []string{
			fmt.Sprintf("%s %s", label.Render("Firewall Engine:"), text.Render(strings.ToUpper(m.report.FirewallType))),
			fmt.Sprintf("%s %s", label.Render("UFW Firewall Status:"), ufwStr),
			fmt.Sprintf("%s %s", label.Render("SSH Root Login:"), rootStr),
			fmt.Sprintf("%s %s", label.Render("SSH Password Auth:"), passStr),
		}

		controls := dim.Render("\nControls: [R] Rerun Audit")
		content = lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			items[0], items[1], items[2], items[3],
			controls,
		)

	case stateFirewall:
		rulesView := label.Render(fmt.Sprintf("Active Rules (%s):", m.report.FirewallType)) + "\n"
		if len(m.report.Rules) == 0 {
			rulesView += dim.Render("No rules configured or firewall inactive.")
		} else {
			for _, r := range m.report.Rules {
				rulesView += fmt.Sprintf("[%2s] %-10s %-15s %s\n", r.ID, r.Action, r.To, r.From)
			}
		}

		controls := dim.Render("\nControls: [a] Add Rule  [d] Delete Rule  [r] Rerun Audit")
		content = lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			rulesView,
			controls,
		)

	case stateAddRule:
		form := lipgloss.JoinVertical(lipgloss.Left,
			label.Render("Add Firewall Rule:"),
			"",
			"Port:    "+m.portInput.View(),
			"Proto:   "+m.protoInput.View(),
			"Action:  "+m.actionInput.View(),
			"",
			dim.Render("[Enter] Next/Submit   [Esc] Cancel"),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, header, "", form)

	case stateDeleteRule:
		form := lipgloss.JoinVertical(lipgloss.Left,
			label.Render("Delete Firewall Rule:"),
			"",
			"Rule ID: "+m.idInput.View(),
			"",
			dim.Render("[Enter] Submit   [Esc] Cancel"),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, header, "", form)

	case statePorts:
		portsView := label.Render("Open Ports & Owning Processes:") + "\n\n"
		portsView += dim.Render(fmt.Sprintf("%-8s %-25s %-20s", "PROTO", "ADDRESS:PORT", "PROCESS")) + "\n"
		for _, p := range m.report.Ports {
			addrStr := p.Address + ":" + p.Port
			if p.Address == "0.0.0.0" {
				addrStr = red.Render(addrStr)
			} else if p.Address == "*" || p.Address == "::" {
				addrStr = yellow.Render(addrStr)
			}
			portsView += fmt.Sprintf("%-8s %-25s %-20s\n", p.Protocol, addrStr, p.Process)
		}

		controls := dim.Render("\nControls: [r] Rerun Audit")
		content = lipgloss.JoinVertical(lipgloss.Left, header, "", portsView, controls)

	case stateFail2Ban:
		view := label.Render("Fail2Ban Status:") + "\n"
		if len(m.report.Jails) == 0 {
			view += dim.Render("Fail2Ban is not active or no jails found.") + "\n\n"
		} else {
			for _, j := range m.report.Jails {
				banned := strings.Join(j.BannedIPs, ", ")
				if banned == "" {
					banned = "None"
				}
				view += fmt.Sprintf("- %s: Banned: %s\n", text.Bold(true).Render(j.Name), banned)
			}
			view += "\n"
		}

		view += label.Render("Failed SSH Logins (Last 10 Days):") + "\n"
		if len(m.report.Logins) == 0 {
			view += dim.Render("No failed logins found.") + "\n"
		} else {
			view += dim.Render(fmt.Sprintf("%-6s %-16s %s", "COUNT", "IP", "LAST ATTEMPT")) + "\n"
			for i, l := range m.report.Logins {
				if i >= 8 {
					break // max display 8 to fit screen
				}
				view += fmt.Sprintf("%-6d %-16s %s\n", l.Count, l.IP, l.LastTime)
			}
		}

		controls := dim.Render("\nControls: [b] Ban IP  [u] Unban IP  [r] Rerun Audit")
		content = lipgloss.JoinVertical(lipgloss.Left, header, "", view, controls)

	case stateBanIP, stateUnbanIP:
		title := "Ban IP Address:"
		if m.state == stateUnbanIP {
			title = "Unban IP Address:"
		}
		form := lipgloss.JoinVertical(lipgloss.Left,
			label.Render(title),
			"",
			"IP Address: "+m.ipInput.View(),
			"Jail Name:  "+m.jailInput.View(),
			"",
			dim.Render("[Enter] Next/Submit   [Esc] Cancel"),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, header, "", form)
	}

	return components.Card(content, 75)
}

func (m Model) Title() string { return "Security" }
func (m Model) Icon() string  { return "🛡️" }
