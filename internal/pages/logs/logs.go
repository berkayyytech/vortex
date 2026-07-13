package logs

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	logsengine "main/internal/engine/logs"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	lines        []string
	filtered     []string
	engine       *logsengine.Engine
	paused       bool
	status       string
	
	// Source picker state
	sources      []string
	activeSource int
	pickerOpen   bool
	pickerCursor int
	
	// Scroll state
	scrollOffset int
	
	// Filter state
	filterInput string
	isFiltering bool
}

func New() Model {
	return Model{
		lines:        []string{},
		filtered:     []string{},
		status:       "Connecting to Log Engine...",
		sources:      []string{"System (journalctl)"},
		activeSource: 0,
		pickerOpen:   false,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) applyFilter() {
	m.filtered = []string{}
	if m.filterInput == "" {
		m.filtered = m.lines
		return
	}
	
	f := strings.ToLower(m.filterInput)
	for _, l := range m.lines {
		if strings.Contains(strings.ToLower(l), f) {
			m.filtered = append(m.filtered, l)
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.pickerOpen {
			switch msg.String() {
			case "esc", "q":
				m.pickerOpen = false
			case "up", "k":
				if m.pickerCursor > 0 { m.pickerCursor-- }
			case "down", "j":
				if m.pickerCursor < len(m.sources)-1 { m.pickerCursor++ }
			case "enter":
				m.activeSource = m.pickerCursor
				m.pickerOpen = false
				m.lines = []string{}
				m.filtered = []string{}
				m.scrollOffset = 0
				m.status = fmt.Sprintf("Switched to %s", m.sources[m.activeSource])
			}
			return m, nil
		}

		if m.isFiltering {
			switch msg.String() {
			case "esc", "enter":
				m.isFiltering = false
			case "backspace":
				if len(m.filterInput) > 0 {
					m.filterInput = m.filterInput[:len(m.filterInput)-1]
					m.applyFilter()
				}
			default:
				if len(msg.String()) == 1 {
					m.filterInput += msg.String()
					m.applyFilter()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "p", "P":
			m.paused = !m.paused
		case "up", "k":
			if m.scrollOffset < len(m.filtered)-1 { m.scrollOffset++ }
			m.paused = true // Auto-pause when user scrolls up
		case "down", "j":
			if m.scrollOffset > 0 { m.scrollOffset-- }
		case "pgup":
			m.scrollOffset += 20
			if m.scrollOffset > len(m.filtered)-1 { m.scrollOffset = len(m.filtered)-1 }
			m.paused = true
		case "pgdown":
			m.scrollOffset -= 20
			if m.scrollOffset < 0 { m.scrollOffset = 0 }
		case "s", "S":
			m.pickerOpen = true
			if m.engine != nil {
				return m, func() tea.Msg {
					srcs, err := m.engine.DiscoverSources()
					if err == nil && len(srcs) > 0 {
						return sourcesLoadedMsg(srcs)
					}
					return nil
				}
			}
		case "/":
			m.isFiltering = true
		case "esc":
			if m.filterInput != "" {
				m.filterInput = ""
				m.applyFilter()
			}
		}

	case sourcesLoadedMsg:
		m.sources = []string(msg)
		return m, nil

	case sshlib.ConnectedMsg:
		m.engine = logsengine.NewEngine(msg.Client)
		m.status = "Connected. Press 's' to choose a log source."
		return m, func() tea.Msg {
			srcs, err := m.engine.DiscoverSources()
			if err == nil && len(srcs) > 0 {
				return sourcesLoadedMsg(srcs)
			}
			return nil
		}

	case agent.Payload:
		if !m.paused && m.engine != nil && !m.pickerOpen {
			src := m.sources[m.activeSource]
			
			return m, func() tea.Msg {
				var out string
				var err error
				
				if strings.HasPrefix(src, "[Docker]") {
					out, err = m.engine.FetchDockerLogs(strings.TrimSpace(strings.TrimPrefix(src, "[Docker]")), 100)
				} else if strings.HasPrefix(src, "[Service]") {
					out, err = m.engine.FetchServiceLogs(strings.TrimSpace(strings.TrimPrefix(src, "[Service]")), 100)
				} else if src == "Syslog (/var/log/syslog)" {
					out, err = m.engine.FetchFileLogs("/var/log/syslog", 100)
				} else {
					out, err = m.engine.FetchSystemdLogs(100)
				}
				
				if err != nil {
					return logsLoadedMsg("Error fetching logs: " + err.Error())
				}
				return logsLoadedMsg(out)
			}
		}
		return m, nil
	
	case logsLoadedMsg:
		if !m.paused {
			// We fetched the last 100 lines. Simply replace our buffer to act like a tail (or we can merge)
			// For SSH performance, replacing the buffer with the tail window is extremely fast
			rawLines := strings.Split(string(msg), "\n")
			
			m.lines = []string{}
			for _, l := range rawLines {
				if strings.TrimSpace(l) != "" {
					m.lines = append(m.lines, l)
				}
			}
			
			if len(m.lines) > 5000 {
				m.lines = m.lines[len(m.lines)-5000:]
			}
			m.applyFilter()
			
			if m.scrollOffset > len(m.filtered) {
				m.scrollOffset = len(m.filtered) - 1
			}
		}
		return m, nil
	}
	return m, nil
}

type logsLoadedMsg string
type sourcesLoadedMsg []string

func (m Model) View() string {
	dimColor := theme.Current.Dim
	
	if m.pickerOpen {
		var b strings.Builder
		b.WriteString(components.Title("LOG SOURCES") + "\n\n")
		for i, s := range m.sources {
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if m.pickerCursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(s)))
		}
		b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("\nPress Enter to select, ESC to cancel."))
		return components.Card(b.String(), 90)
	}

	var status string
	if m.paused {
		status = lipgloss.NewStyle().Foreground(theme.Current.Warning).Bold(true).Render("[PAUSED (Auto-scroll disabled)]")
	} else {
		status = lipgloss.NewStyle().Foreground(theme.Current.Success).Render("[LIVE STREAMING]")
	}

	filterUI := ""
	if m.isFiltering || m.filterInput != "" {
		filterUI = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("Filter: ") + m.filterInput
		if m.isFiltering { filterUI += "█" }
		filterUI += "\n\n"
	}

	// Render log lines backwards (tail at bottom)
	var logOutput strings.Builder
	
	// Max lines to show on screen
	displayLines := 30
	
	startIdx := len(m.filtered) - 1 - m.scrollOffset
	if startIdx < 0 { startIdx = 0 }
	
	endIdx := startIdx - displayLines
	if endIdx < 0 { endIdx = -1 }
	
	for i := endIdx + 1; i <= startIdx; i++ {
		if i >= 0 && i < len(m.filtered) {
			line := m.filtered[i]
			
			// Semantic highlighting
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			if strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") || strings.Contains(line, "level=error") {
				style = lipgloss.NewStyle().Foreground(theme.Current.Error)
			} else if strings.Contains(line, "WARN") || strings.Contains(line, "level=warn") {
				style = lipgloss.NewStyle().Foreground(theme.Current.Warning)
			} else if strings.Contains(line, "DEBUG") || strings.Contains(line, "level=debug") {
				style = lipgloss.NewStyle().Foreground(dimColor)
			}
			
			// Truncate to terminal width
			if len(line) > 130 {
				line = line[:127] + "..."
			}
			
			logOutput.WriteString(style.Render(line) + "\n")
		}
	}

	if len(m.filtered) == 0 {
		logOutput.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("No logs found."))
	}

	controls := lipgloss.NewStyle().Foreground(dimColor).Render("\n[s] Change Source  [P] Pause/Resume  [Up/Dn] Scroll  [/] Filter  [ESC] Clear")

	headerTitle := fmt.Sprintf("LOG VIEWER — %s", m.sources[m.activeSource])

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title(headerTitle)+"  "+status,
		filterUI,
		logOutput.String(),
		controls,
	)

	return components.Card(content, 140)
}

func (m Model) IsInputActive() bool {
	return m.isFiltering
}

func (m Model) Title() string { return "Logs" }
func (m Model) Icon() string { return "📜" }
