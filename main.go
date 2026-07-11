package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct{}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			return m, nil
		}

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

	selected := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("> Dashboard")

	menu := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		selected,
		"  Docker",
		"  Services",
		"  Files",
		"  Logs",
		"  SSH",
		"  Settings",
	)

	sidebar := sidebarStyle.Render(menu)

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("Dashboard")

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	stats := card.Render(
		"CPU      ██████░░░░ 38%\n" +
			"RAM      ███░░░░░░ 24%\n" +
			"Disk     █████░░░░ 47%\n\n" +
			"Uptime   16d 4h\n" +
			"OS       Ubuntu 24.04\n" +
			"Kernel   6.8",
	)

	containers := card.Render(
		"Containers\n\n" +
			"✓ nginx\n" +
			"✓ postgres\n" +
			"✓ redis\n" +
			"✓ app",
	)

	logs := card.Render(
		"Recent Activity\n\n" +
			"• nginx restarted\n" +
			"• docker updated\n" +
			"• PM2 restarted\n" +
			"• SSH connected",
	)

	content := contentStyle.Render(
		header + "\n\n" +
			stats + "\n\n" +
			containers + "\n\n" +
			logs,
	)

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
