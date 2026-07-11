package dashboard

import (
	"fmt"
	"strings"

	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
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
	sysStats      stats.SystemStats
	client        *sshlib.Client
	netInfo       *NetworkInfo
	isTesting     bool
	testAnimFrame int
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchStatsCmd, tickCmd())
}

type statsMsg stats.SystemStats
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchStatsCmd() tea.Msg {
	return statsMsg(stats.GetSystemStats())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case sshlib.ConnectedMsg:
		m.client = msg.Client
		return m, nil
	case statsMsg:
		m.sysStats = stats.SystemStats(msg)
		return m, nil
	case agent.Payload:
		m.sysStats = msg.Stats
		return m, nil
	case tickMsg:
		var cmd tea.Cmd
		if m.isTesting {
			m.testAnimFrame++
		} else {
			cmd = fetchStatsCmd
		}
		return m, tea.Batch(cmd, tickCmd())
	case speedTestResultMsg:
		m.isTesting = false
		m.netInfo = msg
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "t", "T":
			if !m.isTesting && m.client != nil {
				m.isTesting = true
				m.netInfo = nil
				return m, runSpeedTest(m.client)
			}
		}
	}
	return m, nil
}

type speedTestResultMsg *NetworkInfo

func runSpeedTest(c *sshlib.Client) tea.Cmd {
	return func() tea.Msg {
		script := `
		if command -v curl >/dev/null 2>&1; then
			IP=$(curl -s --max-time 3 ipinfo.io/ip)
			ISP=$(curl -s --max-time 3 ipinfo.io/org | cut -d' ' -f2-)
			PING=$(ping -c 3 1.1.1.1 | tail -1 | awk -F '/' '{print $4}')
			DOWN_BPS=$(curl -s -w "%{speed_download}" -m 10 -o /dev/null https://speed.cloudflare.com/__down?bytes=25000000)
			
			if [ -z "$ISP" ]; then ISP="Unknown"; fi
			if [ -z "$IP" ]; then IP="Unknown"; fi
			if [ -z "$PING" ]; then PING="N/A"; fi
			
			echo "$ISP|$IP|${PING}ms|$DOWN_BPS"
		else
			echo "error"
		fi
		`
		out, err := c.Run(script)
		if err != nil || strings.TrimSpace(out) == "error" {
			return speedTestResultMsg(&NetworkInfo{Download: "Error", Ping: "N/A", ISP: "N/A", IP: "N/A"})
		}
		
		parts := strings.Split(strings.TrimSpace(out), "|")
		if len(parts) != 4 {
			return speedTestResultMsg(&NetworkInfo{Download: "Failed", Ping: "N/A", ISP: "N/A", IP: "N/A"})
		}
		
		var bps float64
		fmt.Sscanf(parts[3], "%f", &bps)
		mbps := (bps * 8) / 1000000
		
		return speedTestResultMsg(&NetworkInfo{
			ISP:      parts[0],
			IP:       parts[1],
			Ping:     parts[2],
			Download: fmt.Sprintf("%.1f Mbps", mbps),
		})
	}
}

func (m Model) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(theme.Current.Text).Width(8)
	
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	
	var netView string
	if m.isTesting {
		status := lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(spinner[m.testAnimFrame%len(spinner)] + " Testing connection...")
		netView = fmt.Sprintf("%s %s\n\n", lipgloss.NewStyle().Foreground(theme.Current.Text).Width(15).Render("Status"), status)
	} else if m.netInfo != nil {
		netView = fmt.Sprintf("%s %s\n%s %s\n%s %s\n%s %s\n\n",
			lipgloss.NewStyle().Foreground(theme.Current.Text).Width(15).Render("ISP"), lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.netInfo.ISP),
			lipgloss.NewStyle().Foreground(theme.Current.Text).Width(15).Render("IP Address"), lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(m.netInfo.IP),
			lipgloss.NewStyle().Foreground(theme.Current.Text).Width(15).Render("Latency (Ping)"), lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(m.netInfo.Ping),
			lipgloss.NewStyle().Foreground(theme.Current.Text).Width(15).Render("Down Speed"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true).Render("↓ "+m.netInfo.Download),
		)
	} else {
		netView = lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("Press [T] to run an internet speed test\n\n")
	}

	content := strings.Join([]string{
		components.Title("HARDWARE UTILIZATION"),
		"",
		labelStyle.Render("CPU") + " " + components.ProgressBar(m.sysStats.CPUPercent, 30),
		labelStyle.Render("RAM") + " " + components.ProgressBar(m.sysStats.RAMPercent, 30),
		labelStyle.Render("Disk") + " " + components.ProgressBar(m.sysStats.DiskPercent, 30),
		"",
		components.Title("NETWORK METRICS"),
		"",
		netView,
		components.Title("SYSTEM INFORMATION"),
		"",
		fmt.Sprintf("%s %s", labelStyle.Render("Uptime"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.Uptime)),
		fmt.Sprintf("%s %s", labelStyle.Render("OS"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.OS)),
		fmt.Sprintf("%s %s", labelStyle.Render("Kernel"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.Kernel)),
		"",
		lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("Controls: [T] Run Network Speed Test"),
	}, "\n")

	return components.Card(content, 60)
}

func (m Model) Title() string {
	return "Dashboard"
}

func (m Model) Icon() string {
	return "📊"
}
