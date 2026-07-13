package apps

import (
	"fmt"

	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	appengine "main/internal/engine/apps"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type SortMode int

const (
	SortCPU SortMode = iota
	SortMem
	SortPID
)

type Model struct {
	apps          []appengine.App
	filteredApps  []appengine.App
	cursor        int
	engine        *appengine.Engine
	status        string
	sortMode      SortMode
	filterInput   string
	isFiltering   bool
	killConfirm   bool
}

func New() Model {
	return Model{
		apps:          nil,
		filteredApps:  nil,
		cursor:        0,
		status:        "Connecting to Process Engine...",
		sortMode:      SortCPU,
		filterInput:   "",
		isFiltering:   false,
		killConfirm:   false,
	}
}

func (m *Model) applyFilterAndSort() {
	m.filteredApps = []appengine.App{}
	for _, app := range m.apps {
		// Hide vortex internal processes
		nameL := strings.ToLower(app.Name)
		if strings.Contains(nameL, "top -c -b -d") || strings.Contains(nameL, "awk ") || strings.Contains(nameL, "vps-manager") || strings.Contains(nameL, "vortex") || strings.Contains(nameL, "ping -c 3") || strings.Contains(nameL, "tail -1") || strings.Contains(nameL, "head -n") {
			continue
		}
		if m.filterInput != "" && !strings.Contains(strings.ToLower(app.Name), strings.ToLower(m.filterInput)) && !strings.Contains(strings.ToLower(app.User), strings.ToLower(m.filterInput)) {
			continue
		}
		m.filteredApps = append(m.filteredApps, app)
	}

	sort.Slice(m.filteredApps, func(i, j int) bool {
		switch m.sortMode {
		case SortCPU:
			return m.filteredApps[i].CPU > m.filteredApps[j].CPU
		case SortMem:
			return m.filteredApps[i].Mem > m.filteredApps[j].Mem
		case SortPID:
			return m.filteredApps[i].PID < m.filteredApps[j].PID
		}
		return false
	})
}

func (m Model) Init() tea.Cmd { return nil }

type appsLoadedMsg []appengine.App

func fetchApps(engine *appengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		apps, err := engine.DetectApps()
		if err != nil {
			return nil // ignore silent failure
		}
		return appsLoadedMsg(apps)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case appsLoadedMsg:
		m.apps = msg
		m.applyFilterAndSort()
		m.status = fmt.Sprintf("Monitoring %d processes.", len(m.filteredApps))
		if m.cursor >= len(m.filteredApps) {
			m.cursor = 0
		}
		return m, nil

	case tea.KeyMsg:
		if m.killConfirm {
			switch msg.String() {
			case "y", "Y", "enter":
				m.killConfirm = false
				if len(m.filteredApps) > 0 && m.engine != nil {
					return m, func() tea.Msg {
						m.engine.KillProcess(m.filteredApps[m.cursor].PID)
						return fetchApps(m.engine)()
					}
				}
			default: // any other key cancels
				m.killConfirm = false
				return m, nil
			}
		}

		if m.isFiltering {
			switch msg.String() {
			case "esc", "enter":
				m.isFiltering = false
			case "backspace":
				if len(m.filterInput) > 0 {
					m.filterInput = m.filterInput[:len(m.filterInput)-1]
					m.applyFilterAndSort()
					if m.cursor >= len(m.filteredApps) { m.cursor = 0 }
				}
			default:
				if len(msg.String()) == 1 {
					m.filterInput += msg.String()
					m.applyFilterAndSort()
					if m.cursor >= len(m.filteredApps) { m.cursor = 0 }
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
			if m.cursor < len(m.filteredApps)-1 {
				m.cursor++
			}
		case "s", "S":
			if len(m.filteredApps) > 0 && m.engine != nil {
				return m, func() tea.Msg {
					m.engine.StopProcess(m.filteredApps[m.cursor].PID)
					return fetchApps(m.engine)()
				}
			}
		case "K": // Shift+K for forced kill
			if len(m.filteredApps) > 0 {
				m.killConfirm = true
			}
		case "r", "R":
			m.status = "Refreshing processes..."
			return m, fetchApps(m.engine)
		case "c":
			m.sortMode = SortCPU
			m.applyFilterAndSort()
		case "m":
			m.sortMode = SortMem
			m.applyFilterAndSort()
		case "p":
			m.sortMode = SortPID
			m.applyFilterAndSort()
		case "/":
			m.isFiltering = true
		}

	case sshlib.ConnectedMsg:
		m.engine = appengine.NewEngine(msg.Client)
		m.status = "Scanning ports and runtimes..."
		return m, fetchApps(m.engine)
	}
	return m, nil
}

func (m Model) View() string {
	var items string
	if m.apps == nil {
		items = lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.status)
	} else if len(m.filteredApps) == 0 {
		items = "No processes match the current filter."
	} else {
		header := lipgloss.NewStyle().Foreground(theme.Current.Dim).Bold(true).Render(
			fmt.Sprintf("  %-8s %-10s %-8s %-8s %-12s %s\n", "PID", "USER", "CPU%", "MEM%", "TYPE", "COMMAND"),
		)
		items += header

		for i, app := range m.filteredApps {
			cursor := "  "
			
			rowColor := theme.Current.Text
			if app.User == "root" {
				rowColor = lipgloss.Color("203") // distinct dim red for root processes
			}
			
			style := lipgloss.NewStyle().Foreground(rowColor)
			
			cpuColor := theme.Current.Success
			if app.CPU > 50 { cpuColor = theme.Current.Warning }
			if app.CPU > 80 { cpuColor = theme.Current.Error }
			
			memColor := theme.Current.Success
			if app.Mem > 50 { memColor = theme.Current.Warning }
			if app.Mem > 80 { memColor = theme.Current.Error }

			cpuStr := lipgloss.NewStyle().Foreground(cpuColor).Render(fmt.Sprintf("%-8.1f", app.CPU))
			memStr := lipgloss.NewStyle().Foreground(memColor).Render(fmt.Sprintf("%-8.1f", app.Mem))
			
			cmdStr := app.Name
			if len(cmdStr) > 50 {
				cmdStr = cmdStr[:47] + "..."
			}
			
			userStr := app.User
			if len(userStr) > 10 { userStr = userStr[:10] }
			typeStr := "[" + app.Runtime + "]"

			var row string
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("232")).Background(theme.Current.Primary).Bold(true)
				row = fmt.Sprintf("%-8s %-10s %-8.1f %-8.1f %-12s %s", app.PID, userStr, app.CPU, app.Mem, typeStr, cmdStr)
				row = style.Render(row)
			} else {
				row = fmt.Sprintf("%-8s %-10s %s %s %-12s %s", 
					lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(app.PID),
					lipgloss.NewStyle().Foreground(rowColor).Render(userStr),
					cpuStr,
					memStr,
					lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(typeStr),
					lipgloss.NewStyle().Foreground(theme.Current.Text).Render(cmdStr),
				)
			}
			
			items += cursor + row + "\n"
		}
	}

	filterUI := ""
	if m.isFiltering || m.filterInput != "" {
		filterUI = lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("Filter: ") + m.filterInput
		if m.isFiltering { filterUI += "█" }
		filterUI += "\n\n"
	}

	confirmUI := ""
	if m.killConfirm {
		appName := m.filteredApps[m.cursor].Name
		if len(appName) > 20 { appName = appName[:20] + "..." }
		confirmUI = lipgloss.NewStyle().Foreground(theme.Current.Error).Bold(true).Render(
			fmt.Sprintf("\n[!] Force kill %s (PID %s)? (y/N)", appName, m.filteredApps[m.cursor].PID),
		)
	}

	sortName := "CPU"
	if m.sortMode == SortMem { sortName = "MEM" }
	if m.sortMode == SortPID { sortName = "PID" }

	title := ""
	if m.apps != nil {
		title = fmt.Sprintf("PROCESS MANAGER — Top %d (by %s ▼)", len(m.filteredApps), sortName)
	} else {
		title = "PROCESS MANAGER"
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [c] Sort CPU  [m] Sort MEM  [/] Filter  [S] Stop  [Shift+K] Force Kill")

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title(title),
		filterUI,
		items,
		confirmUI,
		controls,
	)

	return components.Card(content, 110)
}

func (m Model) Title() string { return "Processes" }
func (m Model) Icon() string { return "⚙" }
