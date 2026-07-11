package backup

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	backupengine "main/internal/engine/backup"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	jobs   []backupengine.BackupJob
	cursor int
	engine *backupengine.Engine
	status string
}

func New() Model {
	return Model{status: "Connecting..."}
}

func (m Model) Init() tea.Cmd { return nil }

type backupsLoadedMsg []backupengine.BackupJob
type backupCompleteMsg struct {
	job *backupengine.BackupJob
	err error
}

func fetchBackups(engine *backupengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		jobs, _ := engine.ListBackups()
		return backupsLoadedMsg(jobs)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case backupsLoadedMsg:
		m.jobs = msg
		m.status = "Idle."
		if m.cursor >= len(m.jobs) && len(m.jobs) > 0 {
			m.cursor = len(m.jobs) - 1
		}
		return m, nil

	case backupCompleteMsg:
		if msg.err != nil {
			m.status = "Backup Failed: " + msg.err.Error()
		} else {
			m.status = "Backup Successful: " + msg.job.ID
		}
		return m, fetchBackups(m.engine)

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.jobs)-1 {
				m.cursor++
			}
		case "b", "B":
			// Create a generic backup of /etc for demonstration purposes
			if m.engine != nil {
				m.status = "Creating backup of /etc..."
				return m, func() tea.Msg {
					job, err := m.engine.CreateBackup("/etc", "File System")
					return backupCompleteMsg{job: job, err: err}
				}
			}
		case "r", "R":
			m.status = "Scanning for backups..."
			return m, fetchBackups(m.engine)
		}

	case sshlib.ConnectedMsg:
		m.engine = backupengine.NewEngine(msg.Client)
		m.status = "Scanning for backups..."
		return m, fetchBackups(m.engine)
	}
	return m, nil
}

func (m Model) View() string {
	var items string
	if m.jobs == nil || len(m.jobs) == 0 {
		items = "No backups found in /tmp."
	} else {
		for i, job := range m.jobs {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			
			sizeTag := lipgloss.NewStyle().Foreground(theme.Current.Accent).Render("[" + job.Size + "]")
			items += fmt.Sprintf("%s %s %s\n", cursor, sizeTag, style.Render(job.ID))
		}
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [B] Trigger Test Backup (/etc)  [R] Refresh")
	statusBlock := lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("\nStatus: " + m.status)

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("BACKUP MANAGER"),
		items,
		statusBlock,
		controls,
	)

	return components.Card(content, 70)
}

func (m Model) Title() string { return "Backups" }
func (m Model) Icon() string { return "💾" }
