package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	"main/internal/config"
	auditengine "main/internal/engine/audit"
	"main/internal/pages"
	sshlib "main/internal/ssh"
	"main/internal/stats"
	"main/internal/theme"
)

type NetworkInfo struct {
	Download string
	Ping     string
	ISP      string
	IP       string
}

type Model struct {
	payload       agent.Payload
	sysStats      stats.SystemStats
	client        *sshlib.Client
	activeHost    string
	activeUser    string
	netInfo       *NetworkInfo
	isTesting     bool
	testAnimFrame int

	cpuHistory     []float64
	ramHistory     []float64
	diskHistory    []float64
	netDownHistory []float64
	netUpHistory   []float64

	servers        []config.ServerConfig
	width          int
	height         int
}

func New() Model {
	cfg, _ := config.LoadConfig()
	return Model{
		servers: cfg.Servers,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchStatsCmd, tickCmd())
}

type statsMsg stats.SystemStats
type tickMsg time.Time

func tickCmd() tea.Cmd {
	interval := config.CurrentConfig.Monitoring.RefreshInterval
	if interval < 1 {
		interval = 1
	}
	return tea.Tick(time.Second*time.Duration(interval), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchStatsCmd() tea.Msg {
	return statsMsg(stats.GetSystemStats())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case sshlib.ConnectedMsg:
		m.client = msg.Client
		m.activeHost = msg.Host
		m.activeUser = msg.User
		auditengine.GlobalEngine.Append(fmt.Sprintf("SSH login to %s", msg.Host))
		m.isTesting = true
		return m, fetchNetworkMeta(m.client)
	case pages.LogActivityMsg:
		auditengine.GlobalEngine.Append(msg.Message)
		return m, nil
	case statsMsg:
		m.sysStats = stats.SystemStats(msg)
		m.cpuHistory = append(m.cpuHistory, m.sysStats.CPUPercent)
		m.ramHistory = append(m.ramHistory, m.sysStats.RAMPercent)
		m.diskHistory = append(m.diskHistory, m.sysStats.DiskPercent)

		if len(m.cpuHistory) > 20 { m.cpuHistory = m.cpuHistory[1:] }
		if len(m.ramHistory) > 20 { m.ramHistory = m.ramHistory[1:] }
		if len(m.diskHistory) > 20 { m.diskHistory = m.diskHistory[1:] }
		return m, nil
	case agent.Payload:
		m.payload = msg
		m.sysStats = msg.Stats
		m.cpuHistory = append(m.cpuHistory, m.sysStats.CPUPercent)
		m.ramHistory = append(m.ramHistory, m.sysStats.RAMPercent)
		m.diskHistory = append(m.diskHistory, m.sysStats.DiskPercent)

		rxMbps := float64(msg.Network.Bandwidth.CurrentRx) * 8.0 / 1000000.0
		txMbps := float64(msg.Network.Bandwidth.CurrentTx) * 8.0 / 1000000.0
		m.netDownHistory = append(m.netDownHistory, rxMbps)
		m.netUpHistory = append(m.netUpHistory, txMbps)

		if len(m.cpuHistory) > 20 { m.cpuHistory = m.cpuHistory[1:] }
		if len(m.ramHistory) > 20 { m.ramHistory = m.ramHistory[1:] }
		if len(m.diskHistory) > 20 { m.diskHistory = m.diskHistory[1:] }
		if len(m.netDownHistory) > 20 { m.netDownHistory = m.netDownHistory[1:] }
		if len(m.netUpHistory) > 20 { m.netUpHistory = m.netUpHistory[1:] }

		return m, nil
	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, fetchStatsCmd)
		if m.isTesting {
			m.testAnimFrame++
		}
		
		// Run automated lightweight network poll every 30s
		if time.Now().Second() % 30 == 0 && m.client != nil && !m.isTesting {
			m.isTesting = true
			cmds = append(cmds, fetchNetworkMeta(m.client))
		}
		
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)
	case speedTestResultMsg:
		m.isTesting = false
		m.netInfo = msg
		return m, nil
	case tea.KeyMsg:
		// Manual trigger disabled in favor of automated ping loop
	}
	return m, nil
}

type speedTestResultMsg *NetworkInfo

func fetchNetworkMeta(c *sshlib.Client) tea.Cmd {
	return func() tea.Msg {
		script := `
		if command -v curl >/dev/null 2>&1; then
			IP=$(curl -s --max-time 3 ipinfo.io/ip)
			ISP=$(curl -s --max-time 3 ipinfo.io/org | cut -d' ' -f2-)
			PING=$(ping -c 3 1.1.1.1 | tail -1 | awk -F '/' '{print $4}')
			
			if [ -z "$ISP" ]; then ISP="Unknown"; fi
			if [ -z "$IP" ]; then IP="Unknown"; fi
			if [ -z "$PING" ]; then PING="N/A"; fi
			
			echo "$ISP|$IP|${PING}ms"
		else
			echo "error"
		fi
		`
		out, err := c.Run(script)
		if err != nil || strings.TrimSpace(out) == "error" {
			return speedTestResultMsg(&NetworkInfo{Download: "N/A", Ping: "N/A", ISP: "N/A", IP: "N/A"})
		}
		
		parts := strings.Split(strings.TrimSpace(out), "|")
		if len(parts) != 3 {
			return speedTestResultMsg(&NetworkInfo{Download: "N/A", Ping: "N/A", ISP: "N/A", IP: "N/A"})
		}
		
		return speedTestResultMsg(&NetworkInfo{
			ISP:      parts[0],
			IP:       parts[1],
			Ping:     parts[2],
			Download: "N/A",
		})
	}
}

func (m Model) View() string {
	dim := lipgloss.NewStyle().Foreground(theme.Current.Dim)
	accent := lipgloss.NewStyle().Foreground(theme.Current.Accent)
	primary := lipgloss.NewStyle().Foreground(theme.Current.Primary)
	bold := lipgloss.NewStyle().Bold(true)
	success := lipgloss.NewStyle().Foreground(theme.Current.Success)
	warning := lipgloss.NewStyle().Foreground(theme.Current.Warning)
	errorStyle := lipgloss.NewStyle().Foreground(theme.Current.Error)
	info := lipgloss.NewStyle().Foreground(theme.Current.Info)
	network := lipgloss.NewStyle().Foreground(theme.Current.Network)
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Dim).
		Padding(1, 2)

	// Layout Width Calculations
	usableWidth := m.width - 34
	if usableWidth < 80 {
		usableWidth = 80
	}
	cardWidth := (usableWidth - 8) / 4
	if cardWidth < 22 {
		cardWidth = 22
	}
	halfWidth := (usableWidth - 4) / 2
	if halfWidth < 46 {
		halfWidth = 46
	}

	// 1. Premium Header
	headerStatus := dim.Render("⚪ Not Connected")
	hostName := dim.Render("Unknown")
	if m.client != nil {
		headerStatus = success.Render("🟢 Connected")
		hostName = primary.Render(fmt.Sprintf("%s@%s", m.activeUser, m.activeHost))
	}
	
	osStr := m.sysStats.OS
	if osStr == "" { osStr = "Unknown OS" }
	kernelStr := m.sysStats.Kernel
	if kernelStr == "" { kernelStr = "Unknown Kernel" }
	
	pingStr := "--"
	if m.netInfo != nil { pingStr = m.netInfo.Ping }

	header := fmt.Sprintf("%s    %s    Kernel %s    Ping %s    %s    %s    %s",
		headerStatus,
		primary.Render(osStr),
		primary.Render(kernelStr),
		primary.Render(pingStr),
		hostName,
		primary.Render(m.sysStats.Uptime+" Uptime"),
		accent.Render(time.Now().Format("15:04:05")),
	)

	// 2. Metrics Cards
	getTrend := func(hist []float64, invert bool) (string, lipgloss.Color) {
		if len(hist) < 2 { return "  0.0%", theme.Current.Dim }
		diff := hist[len(hist)-1] - hist[len(hist)-2]
		c := theme.Current.Success
		if diff > 0 {
			if invert { c = theme.Current.Success } else { c = theme.Current.Warning }
			return fmt.Sprintf("▲ +%.1f%%", diff), c
		} else if diff < 0 {
			if invert { c = theme.Current.Warning } else { c = theme.Current.Success }
			return fmt.Sprintf("▼ %.1f%%", diff), c
		}
		return "  0.0%", theme.Current.Dim
	}

	cpuVal := fmt.Sprintf("%3.0f%%", m.sysStats.CPUPercent)
	cpuColor := theme.Current.Success
	if m.sysStats.CPUPercent > float64(config.CurrentConfig.Monitoring.CPUCritical) { cpuColor = theme.Current.Error } else if m.sysStats.CPUPercent > float64(config.CurrentConfig.Monitoring.CPUWarning) { cpuColor = theme.Current.Warning }
	cpuTrend, cpuTrendC := getTrend(m.cpuHistory, false)
	
	cpuCard := boxStyle.Copy().Width(cardWidth).Render(
		bold.Render("CPU") + "\n\n" +
		lipgloss.NewStyle().Foreground(cpuColor).Render(cpuVal) + "  " + lipgloss.NewStyle().Foreground(cpuTrendC).Render(cpuTrend) + "\n\n" +
		components.Sparkline(m.cpuHistory, cardWidth-6, cpuColor),
	)

	ramVal := fmt.Sprintf("%3.0f%%", m.sysStats.RAMPercent)
	ramColor := theme.Current.Success
	if m.sysStats.RAMPercent > float64(config.CurrentConfig.Monitoring.CPUCritical) { ramColor = theme.Current.Error } else if m.sysStats.RAMPercent > float64(config.CurrentConfig.Monitoring.CPUWarning) { ramColor = theme.Current.Warning }
	ramTrend, ramTrendC := getTrend(m.ramHistory, false)
	
	ramCard := boxStyle.Copy().Width(cardWidth).Render(
		bold.Render("Memory") + "\n\n" +
		lipgloss.NewStyle().Foreground(ramColor).Render(ramVal) + "  " + lipgloss.NewStyle().Foreground(ramTrendC).Render(ramTrend) + "\n\n" +
		components.Sparkline(m.ramHistory, cardWidth-6, ramColor),
	)

	diskVal := fmt.Sprintf("%3.0f%%", m.sysStats.DiskPercent)
	diskTrend, diskTrendC := getTrend(m.diskHistory, false)
	diskCard := boxStyle.Copy().Width(cardWidth).Render(
		bold.Render("Disk") + "\n\n" +
		primary.Render(diskVal) + "  " + lipgloss.NewStyle().Foreground(diskTrendC).Render(diskTrend) + "\n\n" +
		components.Sparkline(m.diskHistory, cardWidth-6, theme.Current.Primary),
	)

	netDownStr := "↓ 0.0 MB/s"
	netUpStr := "↑ 0.0 MB/s"
	if len(m.netDownHistory) > 0 {
		lastDown := m.netDownHistory[len(m.netDownHistory)-1] / 8.0
		lastUp := m.netUpHistory[len(m.netUpHistory)-1] / 8.0
		netDownStr = fmt.Sprintf("↓ %.1f MB/s", lastDown)
		netUpStr = fmt.Sprintf("↑ %.1f MB/s", lastUp)
	}
	netCard := boxStyle.Copy().Width(cardWidth).Render(
		bold.Render("Network") + "\n\n" +
		network.Render(netUpStr) + "  " + components.Sparkline(m.netUpHistory, cardWidth-16, theme.Current.Network) + "\n" +
		network.Render(netDownStr) + "  " + components.Sparkline(m.netDownHistory, cardWidth-16, theme.Current.Accent),
	)

	topMetricsRow := lipgloss.JoinHorizontal(lipgloss.Top, cpuCard, "  ", ramCard, "  ", diskCard, "  ", netCard)

	// 3. Health Score
	activeServices, failedServices := 0, 0
	for _, s := range m.payload.Services {
		if s.Status == "running" || s.Status == "active" { activeServices++ } else if s.Status == "failed" { failedServices++ }
	}
	healthScore := 100
	critF := float64(config.CurrentConfig.Monitoring.CPUCritical)
	if m.sysStats.CPUPercent > critF { healthScore -= 10 }
	if m.sysStats.RAMPercent > critF { healthScore -= 10 }
	if m.sysStats.DiskPercent > critF { healthScore -= 20 }
	if failedServices > 0 { healthScore -= 10 }

	healthColor := theme.Current.Success
	healthWord := "Excellent"
	if healthScore < 70 { healthColor = theme.Current.Warning; healthWord = "Warning" }
	if healthScore < 40 { healthColor = theme.Current.Error; healthWord = "Critical" }

	meterBlocks := int((float64(healthScore) / 100.0) * 20.0)
	var meterStr string
	for i := 0; i < 20; i++ {
		if i < meterBlocks { meterStr += "█" } else { meterStr += "░" }
	}
	
	healthContent := fmt.Sprintf("%s\n\nHealth Score\n\n%s  %s\n\n", lipgloss.NewStyle().Foreground(healthColor).Render(meterStr), lipgloss.NewStyle().Foreground(healthColor).Bold(true).Render(fmt.Sprintf("%d / 100", healthScore)), dim.Render(healthWord))
	
	warnF := float64(config.CurrentConfig.Monitoring.CPUWarning)
	if m.sysStats.CPUPercent < warnF { healthContent += success.Render("✓") + " CPU healthy\n" } else { healthContent += warning.Render("⚠") + " CPU high load\n" }
	if m.sysStats.DiskPercent < warnF { healthContent += success.Render("✓") + " Disk healthy\n" } else { healthContent += warning.Render("⚠") + " Disk nearing capacity\n" }
	if failedServices == 0 { healthContent += success.Render("✓") + " Services healthy\n" } else { healthContent += errorStyle.Render("⚠") + " Services failing\n" }
	
	if len(m.payload.Network.Interfaces) > 0 { 
		healthContent += success.Render("✓") + " Network responding\n" 
	} else { 
		healthContent += warning.Render("⚠") + " Network degraded\n" 
	}
	
	if m.payload.Docker.Containers > 0 { 
		healthContent += success.Render("✓") + " Docker engine active" 
	} else { 
		healthContent += dim.Render("○") + " No containers running" 
	}

	healthCard := boxStyle.Copy().Width(halfWidth).Render(
		bold.Render("Server Health") + "\n\n" + healthContent,
	)

	// 4. Quick Status (Widgets)
	publicIP := m.payload.Network.PublicIP
	if publicIP == "" { publicIP = "Fetching..." }
	
	openPorts := ""
	for i, p := range m.payload.Network.Ports {
		if i > 2 {
			openPorts += "..."
			break
		}
		portNum := strings.Split(p.Address, ":")
		if len(portNum) > 1 {
			openPorts += portNum[len(portNum)-1] + " "
		}
	}
	if openPorts == "" { openPorts = "None" }
	
	gw := m.payload.Network.Gateway
	if gw == "" { gw = "Unknown" }

	statusGrid := lipgloss.JoinHorizontal(lipgloss.Top, 
		lipgloss.JoinVertical(lipgloss.Left, 
			dim.Render("Connections:  ") + primary.Render(fmt.Sprintf("%d Estab", m.payload.Network.Connection.Established)),
			dim.Render("TCP / UDP:    ") + primary.Render(fmt.Sprintf("%d / %d", m.payload.Network.Connection.ActiveTCP, m.payload.Network.Connection.ActiveUDP)),
			dim.Render("Gateway:      ") + primary.Render(gw),
			dim.Render("Docker:       ") + primary.Render(fmt.Sprintf("%d Containers", m.payload.Docker.Containers)),
		),
		"    ",
		lipgloss.JoinVertical(lipgloss.Left, 
			dim.Render("Failed Svcs:  ") + errorStyle.Render(fmt.Sprintf("%d", failedServices)),
			dim.Render("Processes:    ") + primary.Render(fmt.Sprintf("%d", len(m.payload.Services))),
			dim.Render("Public IP:    ") + info.Render(publicIP),
			dim.Render("Open Ports:   ") + primary.Render(openPorts),
		),
	)
	
	quickStatusCard := boxStyle.Copy().Width(halfWidth).Render(
		bold.Render("System Overview") + "\n\n" + statusGrid,
	)

	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, quickStatusCard, "  ", healthCard)

	// 5. Activity Feed
	activityContent := ""
	lines := strings.Split(strings.TrimSpace(m.payload.Logs), "\n")
	added := 0
	for i := len(lines) - 1; i >= 0 && added < 4; i-- {
		if lines[i] != "" {
			l := lines[i]
			if len(l) > 60 { l = l[:57] + "..." }
			activityContent += dim.Render(time.Now().Format("15:04")) + "  " + l + "\n"
			added++
		}
	}
	if added == 0 {
		for _, act := range auditengine.GlobalEngine.GetRecent(5) {
			ts := act.Timestamp.Format("15:04")
			activityContent += dim.Render(ts) + "  • " + act.Message + "\n"
		}
	}
	activityCard := boxStyle.Copy().Width(halfWidth).Render(
		bold.Render("Recent Activity") + "\n\n" + activityContent,
	)
	
	// 6. Attention Required
	attentionContent := ""
	if m.sysStats.CPUPercent > 90 { attentionContent += errorStyle.Render("• CPU usage above 90%") + "\n" }
	if m.sysStats.RAMPercent > 90 { attentionContent += errorStyle.Render("• Memory usage above 90%") + "\n" }
	if m.sysStats.DiskPercent > 90 { attentionContent += errorStyle.Render("• Disk usage above 90%") + "\n" }
	if failedServices > 0 { attentionContent += errorStyle.Render(fmt.Sprintf("• %d services failing", failedServices)) + "\n" }
	if m.sysStats.CPUPercent > 50 && m.sysStats.CPUPercent <= 90 { attentionContent += warning.Render("• CPU usage elevated") + "\n" }
	if attentionContent == "" {
		attentionContent = success.Render("✓ All systems operational")
	}

	attentionCard := boxStyle.Copy().Width(halfWidth).Render(
		warning.Render("⚠ Attention Required") + "\n\n" + attentionContent,
	)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, activityCard, "  ", attentionCard)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		topMetricsRow,
		"\n",
		middleRow,
		"\n",
		bottomRow,
	)
}

func (m Model) Title() string {
	return "Mission Control"
}

func (m Model) Icon() string {
	return "🚀"
}
