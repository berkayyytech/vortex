package snapshots

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	snapengine "main/internal/engine/snapshots"
	"main/internal/pages"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

func init() {
	pages.Register(New())
}

type Model struct {
	snapshots []snapengine.Snapshot
	cursor    int
	engine    *snapengine.Engine
	status    string
	inputMode bool
	inputVal  string
}

func New() Model {
	return Model{status: "Connecting..."}
}

func (m Model) Init() tea.Cmd { return nil }

type snapshotsLoadedMsg []snapengine.Snapshot
type actionCompleteMsg struct {
	err error
	msg string
}

func fetchSnapshots(engine *snapengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		snaps, _ := engine.ListSnapshots()
		return snapshotsLoadedMsg(snaps)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapshotsLoadedMsg:
		m.snapshots = msg
		m.status = "Idle."
		if m.cursor >= len(m.snapshots) && len(m.snapshots) > 0 {
			m.cursor = len(m.snapshots) - 1
		}
		return m, nil

	case actionCompleteMsg:
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
		} else {
			m.status = msg.msg
		}
		return m, fetchSnapshots(m.engine)

	case tea.KeyMsg:
		if m.inputMode {
			switch msg.String() {
			case "esc":
				m.inputMode = false
				m.inputVal = ""
				m.status = "Idle."
			case "enter":
				targetFile := strings.TrimSpace(m.inputVal)
				m.inputMode = false
				m.inputVal = ""
				if targetFile != "" && m.engine != nil {
					m.status = "Creating snapshot of " + targetFile + "..."
					return m, func() tea.Msg {
						_, err := m.engine.CreateSnapshot(targetFile)
						return actionCompleteMsg{err: err, msg: "Snapshot created for " + targetFile}
					}
				}
				m.status = "Idle."
			case "backspace":
				if len(m.inputVal) > 0 {
					m.inputVal = m.inputVal[:len(m.inputVal)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.inputVal += msg.String()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.snapshots)-1 {
				m.cursor++
			}
		case "s", "S":
			m.inputMode = true
			m.inputVal = ""
			m.status = "Enter absolute file path to snapshot: "
		case "r", "R":
			if len(m.snapshots) > 0 && m.engine != nil {
				snap := m.snapshots[m.cursor]
				m.status = "Rolling back " + snap.Target + "..."
				return m, func() tea.Msg {
					err := m.engine.Rollback(snap.ID, snap.Target)
					return actionCompleteMsg{err: err, msg: "Rollback successful for " + snap.Target}
				}
			}
		case "d", "D":
			if len(m.snapshots) > 0 && m.engine != nil {
				snap := m.snapshots[m.cursor]
				m.status = "Deleting snapshot..."
				return m, func() tea.Msg {
					err := m.engine.DeleteSnapshot(snap.ID)
					return actionCompleteMsg{err: err, msg: "Snapshot deleted."}
				}
			}
		case "f", "F":
			m.status = "Scanning for snapshots..."
			return m, fetchSnapshots(m.engine)
		}

	case sshlib.ConnectedMsg:
		m.engine = snapengine.NewEngine(msg.Client)
		m.status = "Scanning for config snapshots..."
		return m, fetchSnapshots(m.engine)
	}
	return m, nil
}

func (m Model) View() string {
	var items string
	if m.snapshots == nil || len(m.snapshots) == 0 {
		items = "No config snapshots found in /tmp/vortex_snapshots."
	} else {
		start := 0
		maxLines := 15
		if m.cursor > maxLines/2 {
			start = m.cursor - maxLines/2
		}
		end := start + maxLines
		if end > len(m.snapshots) {
			end = len(m.snapshots)
			start = end - maxLines
			if start < 0 {
				start = 0
			}
		}

		// Header
		items += lipgloss.NewStyle().Foreground(theme.Current.Dim).Bold(true).Render(
			fmt.Sprintf("  %-25s %-20s %s\n", "TARGET FILE", "TIMESTAMP", "SIZE"),
		)

		for i := start; i < end; i++ {
			snap := m.snapshots[i]
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)

			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(theme.Current.Primary).Bold(true)
			}

			target := snap.Target
			if len(target) > 25 {
				target = "..." + target[len(target)-22:]
			}

			timestamp := snap.Timestamp
			if len(timestamp) == 15 {
				// 20060102_150405 -> 2006-01-02 15:04:05
				timestamp = fmt.Sprintf("%s-%s-%s %s:%s:%s",
					timestamp[0:4], timestamp[4:6], timestamp[6:8],
					timestamp[9:11], timestamp[11:13], timestamp[13:15])
			}

			row := fmt.Sprintf("%-25s %-20s %s", target, timestamp, snap.Size)

			if m.cursor == i {
				items += cursor + style.Render(row) + "\n"
			} else {
				items += cursor +
					lipgloss.NewStyle().Foreground(theme.Current.Text).Render(fmt.Sprintf("%-25s", target)) + " " +
					lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(fmt.Sprintf("%-20s", timestamp)) + " " +
					lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(snap.Size) + "\n"
			}
		}
	}

	var statusBlock string
	if m.inputMode {
		statusBlock = lipgloss.NewStyle().Foreground(theme.Current.Warning).Render("\n" + m.status + m.inputVal + "█")
	} else {
		statusBlock = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("\nStatus: " + m.status)
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [S] Manual Snapshot  [R] Rollback  [D] Delete  [F] Refresh")

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("CONFIG CHANGE SNAPSHOTS"),
		lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("Automatically backups previous versions of files before they are modified."),
		"\n",
		items,
		statusBlock,
		controls,
	)

	return components.Card(content, 90)
}

func (m Model) Title() string       { return "Snapshots" }
func (m Model) Icon() string        { return "📸" }
func (m Model) IsInputActive() bool { return m.inputMode }
