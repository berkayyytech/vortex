package security

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	secengine "main/internal/engine/security"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	report *secengine.SecurityReport
	engine *secengine.Engine
	status string
}

func New() Model {
	return Model{status: "Connecting..."}
}

func (m Model) Init() tea.Cmd { return nil }

type auditReportMsg *secengine.SecurityReport

func runAudit(engine *secengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		report, err := engine.RunAudit()
		if err != nil {
			return nil // ignore
		}
		return auditReportMsg(report)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case auditReportMsg:
		m.report = msg
		m.status = "Audit complete."
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			m.status = "Running deep security audit..."
			return m, runAudit(m.engine)
		}

	case sshlib.ConnectedMsg:
		m.engine = secengine.NewEngine(msg.Client)
		m.status = "Running initial security audit..."
		return m, runAudit(m.engine)
	}
	return m, nil
}

func (m Model) View() string {
	if m.report == nil {
		return components.Card(lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.status), 60)
	}

	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	label := lipgloss.NewStyle().Foreground(theme.Current.Text).Width(25)

	var ufwStr string
	if m.report.UFWStatus == "Active" {
		ufwStr = green.Render("Secure (Active)")
	} else {
		ufwStr = red.Render("VULNERABLE (Inactive/Missing)")
	}

	var rootStr string
	if m.report.RootLoginEnabled {
		rootStr = red.Render("VULNERABLE (Enabled)")
	} else {
		rootStr = green.Render("Secure (Disabled)")
	}

	var passStr string
	if m.report.PasswordAuthEnabled {
		passStr = red.Render("VULNERABLE (Passwords Allowed)")
	} else {
		passStr = green.Render("Secure (Key Only)")
	}

	items := []string{
		fmt.Sprintf("%s %s", label.Render("UFW Firewall Status:"), ufwStr),
		fmt.Sprintf("%s %s", label.Render("SSH Root Login:"), rootStr),
		fmt.Sprintf("%s %s", label.Render("SSH Password Auth:"), passStr),
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [R] Rerun Audit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("SECURITY CENTER VULNERABILITY SCAN"),
		"",
		items[0],
		items[1],
		items[2],
		controls,
	)

	return components.Card(content, 65)
}

func (m Model) Title() string { return "Security" }
func (m Model) Icon() string { return "🛡️" }
