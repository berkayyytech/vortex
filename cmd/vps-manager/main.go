package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"main/internal/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	cursor   int
	menu     []string
	sysStats stats.SystemStats
}

func initialModel() model {
	return model{
		menu: []string{
			"Dashboard",
			"Docker",
			"Services",
			"Files",
			"Logs",
			"SSH",
			"Settings",
			"Network",
			"Test",
		},
	}
}

func (m model) Init() tea.Cmd {
	return fetchStatsCmd
}

type statsMsg stats.SystemStats

func fetchStatsCmd() tea.Msg {
	return statsMsg(stats.GetSystemStats())
}

// Tick command to update stats every 5 seconds
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Reusable navigation handler
func (m model) handleNavigation(key string) model {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.menu)-1 {
			m.cursor++
		}

	case "enter":
		m.cursor = m.cursor // Placeholder for future action on selection
	}

	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		// Reuse navigation logic
		m = m.handleNavigation(msg.String())

	case statsMsg:
		m.sysStats = stats.SystemStats(msg)
		return m, tickCmd()

	case tickMsg:
		return m, fetchStatsCmd
	}

	return m, nil
}

func (m model) View() string {

	sidebarStyle := lipgloss.NewStyle().
		Width(22).
		Padding(1, 2).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("63"))

	contentStyle := lipgloss.NewStyle().
		Padding(1, 2)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("VORTEX")

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86"))

	normalStyle := lipgloss.NewStyle()

	var items []string
	items = append(items, title)
	items = append(items, "")

	for i, item := range m.menu {
		if i == m.cursor {
			items = append(items, selectedStyle.Render("> "+item))
		} else {
			items = append(items, normalStyle.Render("  "+item))
		}
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, items...)

	sidebar := sidebarStyle.Render(menu)

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(m.menu[m.cursor])

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	statsView := card.Render(
		strings.Join([]string{
			fmt.Sprintf("CPU      %s", stats.FormatBar(m.sysStats.CPUPercent)),
			fmt.Sprintf("RAM      %s", stats.FormatBar(m.sysStats.RAMPercent)),
			fmt.Sprintf("Disk     %s", stats.FormatBar(m.sysStats.DiskPercent)),
			"",
			fmt.Sprintf("Uptime   %s", m.sysStats.Uptime),
			fmt.Sprintf("OS       %s", m.sysStats.OS),
			fmt.Sprintf("Kernel   %s", m.sysStats.Kernel),
		}, "\n"),
	)

	content := contentStyle.Render(
		header + "\n\n" +
			statsView,
	)

	switch m.cursor {
	case 0:
		// Already handled above
	case 1:
		content = contentStyle.Render(
			header + "\n\n" +
				"Active Containers: 5\n" +
				"Images: 12\n" +
				"Networks: 3\n" +
				"Volumes: 7",
		)
	case 2:
		content = contentStyle.Render(
			header + "\n\n" +
				"Active Services: 8\n" +
				"Failed Services: 1\n" +
				"Inactive Services: 2",
		)
	case 3:
		content = contentStyle.Render(
			header + "\n\n" +
				"File Manager Placeholder\n" +
				"List of files and directories would be displayed here.",
		)
	case 4:
		content = contentStyle.Render(
			header + "\n\n" +
				"Log Viewer Placeholder\n" +
				"Recent logs would be displayed here.",
		)
	case 5:
		content = contentStyle.Render(
			header + "\n\n" +
				"SSH Access Placeholder\n" +
				"SSH connection details would be displayed here.",
		)
	case 6:
		content = contentStyle.Render(
			header + "\n\n" +
				"Settings Placeholder\n" +
				"Configuration options would be displayed here.",
		)
	case 7:
		content = contentStyle.Render(
			header + "\n\n" +
				"Test Placeholder\n" +
				"This is a test section for future features.",
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		content,
	)
}

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
