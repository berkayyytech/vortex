package network

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	netlib "main/internal/network"
)

type Model struct {
	netInfo netlib.NetworkInfo
}

func New() Model {
	return Model{
		netInfo: netlib.NetworkInfo{Status: "Running speedtest (approx 15s)..."},
	}
}

func (m Model) Init() tea.Cmd {
	return fetchNetStatsCmd
}

type netMsg netlib.NetworkInfo

func fetchNetStatsCmd() tea.Msg {
	return netMsg(netlib.GetNetworkStats())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case netMsg:
		m.netInfo = netlib.NetworkInfo(msg)
		return m, nil
	case agent.Payload:
		m.netInfo = msg.Network
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	dimColor := lipgloss.Color("240")
	accentColor := lipgloss.Color("205")

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dimColor).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		MarginBottom(1)

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
			titleCard.Render("CONNECTIONS & SPEEDTEST"),
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

	return netTable + "\n" + connStats
}

func (m Model) Title() string { return "Network" }
func (m Model) Icon() string { return "🌐" }
