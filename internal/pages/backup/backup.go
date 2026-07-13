package backup

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	"main/internal/config"
	backupengine "main/internal/engine/backup"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type State int

const (
	StateList State = iota
	StateConfirmRestore
	StateConfirmDelete
	StateInputBackupTarget
	StateConfigDestPath
	StateConfigSchedule
	StateConfigRetention
)

type Model struct {
	jobs         []backupengine.BackupJob
	cursor       int
	engine       *backupengine.Engine
	status       string
	cfg          config.Config
	storageUsage string
	state        State
	textInput    textinput.Model
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 40

	cfg, _ := config.LoadConfig()

	return Model{
		status:    "Connecting...",
		cfg:       cfg,
		state:     StateList,
		textInput: ti,
	}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

type backupsLoadedMsg []backupengine.BackupJob
type backupCompleteMsg struct {
	job *backupengine.BackupJob
	err error
}
type storageUsageMsg string

func fetchBackups(engine *backupengine.Engine, destPath string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		jobs, _ := engine.ListBackups(destPath)
		return backupsLoadedMsg(jobs)
	}
}

func fetchStorageUsage(engine *backupengine.Engine, destPath string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		usage, _ := engine.GetStorageUsage(destPath)
		return storageUsageMsg(usage)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case backupsLoadedMsg:
		m.jobs = msg
		if m.status == "Scanning for backups..." || m.status == "Idle." || m.status == "Connecting..." {
			m.status = "Idle."
		}
		if m.cursor >= len(m.jobs) && len(m.jobs) > 0 {
			m.cursor = len(m.jobs) - 1
		}
		return m, nil

	case storageUsageMsg:
		m.storageUsage = string(msg)
		return m, nil

	case backupCompleteMsg:
		if msg.err != nil {
			m.status = "Operation Failed: " + msg.err.Error()
		} else {
			if msg.job != nil {
				m.status = "Operation Successful: " + msg.job.ID
			} else {
				m.status = "Operation Successful."
			}
		}
		return m, tea.Batch(fetchBackups(m.engine, m.cfg.Backups.DestPath), fetchStorageUsage(m.engine, m.cfg.Backups.DestPath))

	case tea.KeyMsg:
		if m.state != StateList {
			switch msg.String() {
			case "esc":
				m.state = StateList
				m.status = "Aborted."
				return m, nil
			case "enter":
				val := m.textInput.Value()
				switch m.state {
				case StateConfirmRestore:
					if val == "y" || val == "Y" {
						if len(m.jobs) > 0 && m.engine != nil {
							job := m.jobs[m.cursor]
							m.status = "Restoring " + job.ID + "..."
							m.state = StateList
							return m, func() tea.Msg {
								err := m.engine.RestoreBackup(job.ID)
								return backupCompleteMsg{job: &job, err: err}
							}
						}
					}
					m.state = StateList
					m.status = "Restore aborted."
					return m, nil
				case StateConfirmDelete:
					if val == "y" || val == "Y" {
						if len(m.jobs) > 0 && m.engine != nil {
							job := m.jobs[m.cursor]
							m.status = "Deleting " + job.ID + "..."
							m.state = StateList
							return m, func() tea.Msg {
								err := m.engine.DeleteBackup(job.ID)
								return backupCompleteMsg{job: nil, err: err}
							}
						}
					}
					m.state = StateList
					m.status = "Delete aborted."
					return m, nil
				case StateInputBackupTarget:
					m.state = StateList
					if val != "" && m.engine != nil {
						m.status = "Creating backup of " + val + "..."
						dest := m.cfg.Backups.DestPath
						return m, func() tea.Msg {
							job, err := m.engine.CreateBackup(val, dest, "Manual")
							return backupCompleteMsg{job: job, err: err}
						}
					}
					m.status = "Backup aborted (no target)."
					return m, nil
				case StateConfigDestPath:
					m.cfg.Backups.DestPath = val
					config.SaveConfig(m.cfg)
					m.state = StateList
					m.status = "Config updated. Refreshing..."
					return m, tea.Batch(fetchBackups(m.engine, m.cfg.Backups.DestPath), fetchStorageUsage(m.engine, m.cfg.Backups.DestPath))
				case StateConfigSchedule:
					m.cfg.Backups.Schedule = val
					config.SaveConfig(m.cfg)
					m.state = StateList
					m.status = "Schedule updated."
					return m, nil
				case StateConfigRetention:
					if ret, err := strconv.Atoi(val); err == nil {
						m.cfg.Backups.Retention = ret
						config.SaveConfig(m.cfg)
						m.status = "Retention updated."
					} else {
						m.status = "Invalid retention value."
					}
					m.state = StateList
					return m, nil
				}
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

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
			m.state = StateInputBackupTarget
			m.textInput.Placeholder = "Target path (e.g. /etc)"
			m.textInput.SetValue("")
			m.textInput.Focus()
			return m, textinput.Blink
		case "c", "C":
			m.state = StateConfigSchedule
			m.textInput.Placeholder = "Cron schedule (e.g. 0 2 * * *)"
			m.textInput.SetValue(m.cfg.Backups.Schedule)
			m.textInput.Focus()
			return m, textinput.Blink
		case "p", "P":
			m.state = StateConfigDestPath
			m.textInput.Placeholder = "Destination path"
			m.textInput.SetValue(m.cfg.Backups.DestPath)
			m.textInput.Focus()
			return m, textinput.Blink
		case "t", "T":
			m.state = StateConfigRetention
			m.textInput.Placeholder = "Retention (days)"
			m.textInput.SetValue(fmt.Sprintf("%d", m.cfg.Backups.Retention))
			m.textInput.Focus()
			return m, textinput.Blink
		case "l", "L":
			if len(m.jobs) > 0 {
				m.state = StateConfirmRestore
				m.textInput.Placeholder = "Type 'y' to confirm restore"
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "d", "D":
			if len(m.jobs) > 0 {
				m.state = StateConfirmDelete
				m.textInput.Placeholder = "Type 'y' to confirm delete"
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, textinput.Blink
			}
		case "r", "R":
			m.status = "Scanning for backups..."
			return m, tea.Batch(fetchBackups(m.engine, m.cfg.Backups.DestPath), fetchStorageUsage(m.engine, m.cfg.Backups.DestPath))
		}

	case sshlib.ConnectedMsg:
		m.engine = backupengine.NewEngine(msg.Client)
		m.status = "Scanning for backups..."
		return m, tea.Batch(fetchBackups(m.engine, m.cfg.Backups.DestPath), fetchStorageUsage(m.engine, m.cfg.Backups.DestPath))
	}
	return m, cmd
}

func (m Model) View() string {
	var items string
	if m.jobs == nil || len(m.jobs) == 0 {
		items = "No backups found in " + m.cfg.Backups.DestPath + "."
	} else {
		start := 0
		maxLines := 10
		if m.cursor > maxLines/2 {
			start = m.cursor - maxLines/2
		}
		end := start + maxLines
		if end > len(m.jobs) {
			end = len(m.jobs)
			start = end - maxLines
			if start < 0 {
				start = 0
			}
		}

		for i := start; i < end; i++ {
			job := m.jobs[i]
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.cursor == i {
				cursor = "? "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			
			sizeTag := lipgloss.NewStyle().Foreground(theme.Current.Accent).Render("[" + job.Size + "]")
			items += fmt.Sprintf("%s %s %s\n", cursor, sizeTag, style.Render(job.ID))
		}
	}

	configInfo := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(
		fmt.Sprintf("Dest: %s | Schedule: %s | Retention: %d days | Usage: %s", 
			m.cfg.Backups.DestPath, m.cfg.Backups.Schedule, m.cfg.Backups.Retention, m.storageUsage))

	inputView := ""
	if m.state != StateList {
		prompt := ""
		switch m.state {
		case StateConfirmRestore:
			prompt = lipgloss.NewStyle().Foreground(theme.Current.Error).Bold(true).Render("DANGER: Restore will overwrite files! ")
		case StateConfirmDelete:
			prompt = lipgloss.NewStyle().Foreground(theme.Current.Warning).Bold(true).Render("Delete this backup? ")
		case StateInputBackupTarget:
			prompt = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("Target directory to backup: ")
		case StateConfigDestPath:
			prompt = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("New destination path: ")
		case StateConfigSchedule:
			prompt = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("New cron schedule: ")
		case StateConfigRetention:
			prompt = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("New retention (days): ")
		}
		inputView = "\n\n" + prompt + "\n" + m.textInput.View() + "\n(Press Enter to confirm, Esc to cancel)"
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [B] Backup  [L] Restore  [D] Delete  [R] Refresh\nConfig:   [C] Schedule  [P] Path  [T] Retention")
	statusBlock := lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("\nStatus: " + m.status)

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("BACKUP MANAGER"),
		configInfo,
		"\n"+items,
		inputView,
		statusBlock,
		controls,
	)

	return components.Card(content, 75)
}

func (m Model) Title() string { return "Backups" }
func (m Model) Icon() string { return "??" }
