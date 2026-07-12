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
	success := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dimColor).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		MarginBottom(1)

	// Network Interfaces Table
	ifaceStr := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%-12s %-8s %-16s %-16s %-20s", "INTERFACE", "STATUS", "IPV4", "TYPE", "TRAFFIC (TX/RX)")),
		"────────────────────────────────────────────────────────────────────────────",
	)
	for _, iface := range m.netInfo.Interfaces {
		status := success.Render(iface.Status)
		if iface.Status != "UP" { status = dimColor.Render(iface.Status) }
		
		ip := iface.IPv4
		if ip == "" { ip = "None" }
		if len(ip) > 15 { ip = ip[:15] }

		txMbps := float64(iface.TxRate) * 8 / 1000000.0
		rxMbps := float64(iface.RxRate) * 8 / 1000000.0
		traffic := fmt.Sprintf("%.1f / %.1f Mbps", txMbps, rxMbps)

		ifaceStr = lipgloss.JoinVertical(lipgloss.Left, ifaceStr,
			fmt.Sprintf("%-12s %-18s %-16s %-16s %-20s", iface.Name, status, ip, iface.Type, traffic),
		)
	}

	netTable := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Render(ifaceStr)

	// Ports Table
	portStr := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render(fmt.Sprintf("%-8s %-22s %-12s %-20s", "PROTO", "LOCAL ADDRESS", "STATE", "PROCESS")),
		"────────────────────────────────────────────────────────────────",
	)
	for i, p := range m.netInfo.Ports {
		if i > 8 { // limit to 8 for display
			portStr = lipgloss.JoinVertical(lipgloss.Left, portStr, dimColor.Render("... and more"))
			break
		}
		addr := p.Address
		if len(addr) > 20 { addr = addr[:20] }
		proc := p.Process
		if len(proc) > 20 { proc = proc[:20] }
		if proc == "" { proc = "Unknown" }
		portStr = lipgloss.JoinVertical(lipgloss.Left, portStr,
			fmt.Sprintf("%-8s %-22s %-12s %-20s", p.Protocol, addr, p.State, proc),
		)
	}
	portTable := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1).
		Render(portStr)

	connStats := card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("NETWORK OVERVIEW"),
			fmt.Sprintf("Hostname:    %s", m.netInfo.Hostname),
			fmt.Sprintf("Public IP:   %s", m.netInfo.PublicIP),
			fmt.Sprintf("Private IP:  %s", m.netInfo.PrivateIP),
			fmt.Sprintf("Gateway:     %s", m.netInfo.Gateway),
			"",
			titleCard.Render("CONNECTION STATS"),
			fmt.Sprintf("Active TCP:  %d", m.netInfo.Connection.ActiveTCP),
			fmt.Sprintf("Active UDP:  %d", m.netInfo.Connection.ActiveUDP),
			fmt.Sprintf("Established: %d", m.netInfo.Connection.Established),
			fmt.Sprintf("Errors:      %d", m.netInfo.Connection.Errors),
		),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, netTable, "   ", connStats)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, "\n", portTable)
}

func (m Model) Title() string { return "Network" }
func (m Model) Icon() string { return "🌐" }
