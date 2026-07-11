package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"main/internal/docker"
	"main/internal/network"
	"main/internal/stats"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	cursor   int
	menu     []string
	sysStats    stats.SystemStats
	netInfo     network.NetworkInfo
	dockerStats docker.DockerStats
}

func initialModel() model {
	return model{
		menu: []string{
			"Dashboard",
			"Network",
			"Docker",
			"Services",
			"Files",
			"Logs",
			"SSH",
			"Settings",
			"Test",
		},
		netInfo: network.NetworkInfo{
			Status: "Running speedtest (approx 15s)...",
		},
		dockerStats: docker.DockerStats{
			Status: "Connecting...",
		},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchStatsCmd, fetchNetStatsCmd, fetchDockerStatsCmd)
}

type statsMsg stats.SystemStats
type netMsg network.NetworkInfo
type dockerMsg docker.DockerStats

func fetchStatsCmd() tea.Msg {
	return statsMsg(stats.GetSystemStats())
}

func fetchNetStatsCmd() tea.Msg {
	return netMsg(network.GetNetworkStats())
}

func fetchDockerStatsCmd() tea.Msg {
	return dockerMsg(docker.GetDockerStats())
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

	case netMsg:
		m.netInfo = network.NetworkInfo(msg)
		return m, nil

	case dockerMsg:
		m.dockerStats = docker.DockerStats(msg)
		return m, nil

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
		netTable := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).
			Render(
				lipgloss.JoinVertical(lipgloss.Left,
					lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%-15s %-18s %-20s", "INTERFACE", "IP ADDRESS", "TRAFFIC (TX/RX)")),
					"────────────────────────────────────────────────────────",
					fmt.Sprintf("%-15s %-18s %-20s", "eth0", "192.168.1.45", "1.2 GB / 4.5 GB"),
					fmt.Sprintf("%-15s %-18s %-20s", "lo", "127.0.0.1", "12 MB / 12 MB"),
					fmt.Sprintf("%-15s %-18s %-20s", "docker0", "172.17.0.1", "340 MB / 150 MB"),
				),
			)

		connStats := card.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("CONNECTIONS & SPEEDTEST"),
				"",
				fmt.Sprintf("Server:      %s", m.netInfo.ServerName),
				fmt.Sprintf("Ping:        %s", m.netInfo.Ping),
				fmt.Sprintf("Download:    %.2f Mbps", m.netInfo.Download),
				fmt.Sprintf("Upload:      %.2f Mbps", m.netInfo.Upload),
				fmt.Sprintf("Status:      %s", m.netInfo.Status),
				"",
				"Active TCP:  42",
				"Listening:   22 (SSH), 80 (HTTP), 443 (HTTPS)",
			),
		)

		content = contentStyle.Render(
			header + "\n\n" +
				netTable + "\n\n" +
				connStats,
		)
	case 2:
		content = contentStyle.Render(
			header + "\n\n" +
				fmt.Sprintf("Status:             %s\n\n", m.dockerStats.Status) +
				fmt.Sprintf("Active Containers:  %d\n", m.dockerStats.Containers) +
				fmt.Sprintf("Images:             %d\n", m.dockerStats.Images) +
				fmt.Sprintf("Networks:           %d\n", m.dockerStats.Networks) +
				fmt.Sprintf("Volumes:            %d", m.dockerStats.Volumes),
		)
	case 3:
		content = contentStyle.Render(
			header + "\n\n" +
				"Active Services: 8\n" +
				"Failed Services: 1\n" +
				"Inactive Services: 2",
		)
	case 4:
		content = contentStyle.Render(
			header + "\n\n" +
				"File Manager Placeholder\n" +
				"List of files and directories would be displayed here.",
		)
	case 5:
		content = contentStyle.Render(
			header + "\n\n" +
				"Log Viewer Placeholder\n" +
				"Recent logs would be displayed here.",
		)
	case 6:
		content = contentStyle.Render(
			header + "\n\n" +
				"SSH Access Placeholder\n" +
				"SSH connection details would be displayed here.",
		)
	case 7:
		content = contentStyle.Render(
			header + "\n\n" +
				"Settings Placeholder\n" +
				"Configuration options would be displayed here.",
		)
	case 8:
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
