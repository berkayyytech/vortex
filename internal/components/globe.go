package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"main/internal/agent"
	"main/internal/config"
	"main/internal/theme"
)

type Globe struct {
	Active        bool
	Frame         int
	SelectedIdx   int
	Servers       []config.ServerConfig
	ActiveHost    string
	IsEntering    bool
	EnterProgress int
}

func NewGlobe() Globe {
	cfg, _ := config.LoadConfig()
	return Globe{
		Active:      false,
		Frame:       0,
		SelectedIdx: 0,
		Servers:     cfg.Servers,
	}
}

type TickGlobeMsg time.Time

func TickGlobe() tea.Cmd {
	return tea.Tick(time.Millisecond*150, func(t time.Time) tea.Msg {
		return TickGlobeMsg(t)
	})
}

func (g *Globe) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TickGlobeMsg:
		if g.Active {
			g.Frame++
			if g.IsEntering && g.EnterProgress < 10 {
				g.EnterProgress++
			} else {
				g.IsEntering = false
			}
			return TickGlobe()
		}
	case tea.KeyMsg:
		if !g.Active {
			return nil
		}
		switch msg.String() {
		case "up":
			if g.SelectedIdx > 0 {
				g.SelectedIdx--
			} else {
				g.SelectedIdx = len(g.Servers) - 1
			}
		case "down":
			if g.SelectedIdx < len(g.Servers)-1 {
				g.SelectedIdx++
			} else {
				g.SelectedIdx = 0
			}
		}
	}
	return nil
}

func renderCore(frame int, online int, total int, tx float64, rx float64, failedServices int, isSelected bool) string {
	accent := theme.Current.Accent
	color := accent
	ratio := 1.0
	if total > 0 { ratio = float64(online) / float64(total) }
	
	healthScore := 100
	if failedServices > 0 { healthScore -= 20 }
	
	if healthScore < 100 || ratio < 1.0 { color = theme.Current.Warning }
	if ratio <= 0.5 { color = lipgloss.Color("196") }
	if ratio <= 0.25 { color = theme.Current.Dim }

	coreStyle := lipgloss.NewStyle().Foreground(color).Bold(true)

	txStr := fmt.Sprintf("↑%.1fM", tx)
	rxStr := fmt.Sprintf("↓%.1fM", rx)
	if tx < 1 && rx < 1 {
		txStr = fmt.Sprintf("↑%.0fK", tx*1000)
		rxStr = fmt.Sprintf("↓%.0fK", rx*1000)
	}
	statsStr1 := fmt.Sprintf("%s %s", txStr, rxStr)
	statsStr2 := fmt.Sprintf("%d/%d online", online, total)
	
	if len(statsStr1) > 12 {
		statsStr1 = statsStr1[:12]
	} else {
		pad1 := (12 - len(statsStr1)) / 2
		statsStr1 = strings.Repeat(" ", pad1) + statsStr1 + strings.Repeat(" ", 12-pad1-len(statsStr1))
	}
	
	if len(statsStr2) > 12 {
		statsStr2 = statsStr2[:12]
	} else {
		pad2 := (12 - len(statsStr2)) / 2
		statsStr2 = strings.Repeat(" ", pad2) + statsStr2 + strings.Repeat(" ", 12-pad2-len(statsStr2))
	}

	lines := []string{
		"  .────────────.  ",
		" ╱              ╲ ",
		"│      CORE      │",
		fmt.Sprintf("│  %s  │", statsStr1),
		fmt.Sprintf("│  %s  │", statsStr2),
		" ╲              ╱ ",
		"  `────────────`  ",
	}

	speed := 4
	if tx+rx > 50 { speed = 2 }
	if tx+rx > 200 { speed = 1 }
	if online == 0 { speed = 10 }
	
	ringStep := (frame / speed) % 3
	
	var out []string
	for idx, l := range lines {
		lRing := ""
		rRing := ""
		for r := 2; r >= 0; r-- {
			lColor := theme.Current.Dim
			rColor := theme.Current.Dim
			if r == ringStep && online > 0 {
				rColor = color
				lColor = color
			}
			if isSelected && r == (frame/2)%3 {
				lColor = theme.Current.Text
			}
			
			charL, charR := "(", ")"
			if idx == 0 || idx == len(lines)-1 { charL, charR = " ", " " } else if idx == 1 { charL, charR = "╱", "╲" } else if idx == len(lines)-2 { charL, charR = "╲", "╱" }
			
			lRing = lRing + lipgloss.NewStyle().Foreground(lColor).Render(charL)
			rRing = rRing + lipgloss.NewStyle().Foreground(rColor).Render(charR)
		}
		
		corePart := coreStyle.Render(l)
		if isSelected && frame%4 < 2 {
			corePart = lipgloss.NewStyle().Foreground(theme.Current.Text).Bold(true).Render(l)
		}
		
		out = append(out, lRing + corePart + rRing)
	}
	
	return strings.Join(out, "\n")
}

func (g *Globe) View(width, height int, payload agent.Payload, activeHost string, isFetching bool) string {
	accent := theme.Current.Accent
	dim := theme.Current.Dim
	text := theme.Current.Text
	success := theme.Current.Success
	warning := theme.Current.Warning
	danger := lipgloss.Color("196")

	g.ActiveHost = activeHost

	// Entrance Animation Mask
	if g.IsEntering && g.EnterProgress < 2 {
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, "Initializing Mission Control Core...")
	}

	// Calculate simulated traffic intensity
	trafficIntensity := 0.0
	var currentRx, currentTx uint64
	currentRx = payload.Network.Bandwidth.CurrentRx
	currentTx = payload.Network.Bandwidth.CurrentTx
	totalMbps := float64(currentRx+currentTx) * 8 / 1000000.0
	trafficIntensity = totalMbps
	if trafficIntensity > 100 { trafficIntensity = 100 }

	// Pre-calculate infrastructure stats
	failedServices := 0
	for _, s := range payload.Services {
		if s.Status == "failed" { failedServices++ }
	}
	onlineCount := 0
	if activeHost != "" { onlineCount = 1 }

	txMbps := float64(currentTx) * 8 / 1000000.0
	rxMbps := float64(currentRx) * 8 / 1000000.0
	isSelectedActive := false
	if len(g.Servers) > 0 && g.SelectedIdx < len(g.Servers) {
		isSelectedActive = g.Servers[g.SelectedIdx].Host == activeHost
	}

	// 1. CORE CENTER
	coreArt := renderCore(g.Frame, onlineCount, len(g.Servers), txMbps, rxMbps, failedServices, isSelectedActive)

	// 2. LEFT PANEL (Topology & Node List)
	var nodes []string
	
	avgCPU := payload.Stats.CPUPercent
	avgRAM := payload.Stats.RAMPercent
	health := "99%"
	if avgCPU > 90 || avgRAM > 90 { health = "80%" }
	summaryStr := fmt.Sprintf("Online: %d   Offline: %d   Warnings: %d\nAvg CPU: %.0f%%   Avg RAM: %.0f%%\nBandwidth: %.1f Mbps   Health: %s", onlineCount, len(g.Servers)-onlineCount, failedServices, avgCPU, avgRAM, totalMbps, health)
	summaryBox := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(dim).Padding(1).Render(summaryStr)

	// Determine connection weight based on bandwidth
	lineChar := "─"
	if trafficIntensity > 5 { lineChar = "═" }
	if trafficIntensity > 50 { lineChar = "█" }

	for i, srv := range g.Servers {
		style := lipgloss.NewStyle().Foreground(dim)
		indicator := lipgloss.NewStyle().Foreground(dim).Render("⚪")
		cpuRAM := ""
		
		isActiveSrv := srv.Host == activeHost
		if isActiveSrv {
			indicator = lipgloss.NewStyle().Foreground(success).Render("🟢")
			style = lipgloss.NewStyle().Foreground(success)
			if payload.Stats.CPUPercent > 80 || payload.Stats.RAMPercent > 80 {
				indicator = lipgloss.NewStyle().Foreground(warning).Render("🟡")
				style = lipgloss.NewStyle().Foreground(warning)
			}
			if payload.Stats.CPUPercent > 95 {
				indicator = lipgloss.NewStyle().Foreground(danger).Render("🔴")
				style = lipgloss.NewStyle().Foreground(danger)
			}
			cpuRAM = lipgloss.NewStyle().Foreground(dim).Render(fmt.Sprintf(" (CPU %.0f%% RAM %.0f%%)", payload.Stats.CPUPercent, payload.Stats.RAMPercent))
		}

		cursor := "  "
		if i == g.SelectedIdx {
			cursor = lipgloss.NewStyle().Foreground(accent).Render("▶ ")
			style = style.Copy().Bold(true).Foreground(text)
		}

		name := srv.Name
		if name == "" { name = srv.Host }
		if len(name) > 12 { name = name[:12] }
		
		nameStr := fmt.Sprintf("%-12s", name)

		// Animation Logic
		animLine := ""
		spokeLength := 12
		
		lineColor := dim
		if isActiveSrv { lineColor = theme.Current.Primary }
		if i == g.SelectedIdx { lineColor = theme.Current.Text } // Brighten when selected

		if isActiveSrv {
			// Traffic intensity affects speed
			speed := 4
			if trafficIntensity > 50 { speed = 2 }
			if trafficIntensity > 100 { speed = 1 }
			
			packetPos := (g.Frame / speed) % spokeLength
			
			// Build the line with a moving packet
			spoke := ""
			for p := 0; p < spokeLength; p++ {
				if p == packetPos {
					spoke += "●"
				} else {
					spoke += lineChar
				}
			}
			animLine = lipgloss.NewStyle().Foreground(lineColor).Render(spoke + "►")
		} else {
			spoke := strings.Repeat("─", spokeLength)
			animLine = lipgloss.NewStyle().Foreground(lineColor).Render(spoke + "►")
		}
		
		nodes = append(nodes, fmt.Sprintf("%s%s %s%s %s", cursor, indicator, style.Render(nameStr), cpuRAM, animLine))
	}

	leftPanel := lipgloss.NewStyle().
		Width(48).
		Padding(1, 0, 1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(text).Render("INFRASTRUCTURE TOPOLOGY"),
			"",
			summaryBox,
			"",
			strings.Join(nodes, "\n\n"),
		))

	// 3. RIGHT PANEL (Telemetry for Selected Node)
	infoContent := ""
	if len(g.Servers) > 0 && g.Servers[g.SelectedIdx].Host == activeHost {
		// System
		sysGroup := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("SYSTEM"),
			lipgloss.NewStyle().Foreground(dim).Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━"),
			fmt.Sprintf("%-12s %s", "Hostname:", payload.Network.Hostname),
			fmt.Sprintf("%-12s %s", "OS:", payload.Stats.OS),
			fmt.Sprintf("%-12s %s", "Kernel:", payload.Stats.Kernel),
			fmt.Sprintf("%-12s %s", "Uptime:", payload.Stats.Uptime),
		)
		
		// Resources
		resGroup := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("RESOURCES"),
			lipgloss.NewStyle().Foreground(dim).Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━"),
			fmt.Sprintf("%-12s %.1f%%", "CPU:", payload.Stats.CPUPercent),
			fmt.Sprintf("%-12s %.1f%%", "Memory:", payload.Stats.RAMPercent),
			fmt.Sprintf("%-12s %.1f%%", "Disk:", payload.Stats.DiskPercent),
		)

		// Network
		netGroup := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("NETWORK"),
			lipgloss.NewStyle().Foreground(dim).Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━"),
			fmt.Sprintf("%-12s %s", "Public IP:", payload.Network.PublicIP),
			fmt.Sprintf("%-12s %d", "Interfaces:", len(payload.Network.Interfaces)),
			fmt.Sprintf("%-12s %.2f Mbps", "Bandwidth:", totalMbps),
		)

		// Services
		svcGroup := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("SERVICES"),
			lipgloss.NewStyle().Foreground(dim).Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━"),
			fmt.Sprintf("%-12s %d", "Docker:", payload.Docker.Containers),
			fmt.Sprintf("%-12s %d", "Processes:", len(payload.Services)),
		)
		
		// Activity Feed
		lines := strings.Split(strings.TrimSpace(payload.Logs), "\n")
		var recentLogs []string
		for i := len(lines) - 1; i >= 0 && len(recentLogs) < 3; i-- {
			if lines[i] != "" {
				l := lines[i]
				if len(l) > 42 { l = l[:39] + "..." }
				recentLogs = append(recentLogs, lipgloss.NewStyle().Foreground(dim).Render(l))
			}
		}
		if len(recentLogs) == 0 {
			recentLogs = append(recentLogs, fmt.Sprintf("%s 🌐 Telemetry heartbeat", time.Now().Format("15:04:05")))
		}

		activity := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(text).Bold(true).Render("ACTIVITY FEED"),
			strings.Join(recentLogs, "\n"),
		)

		infoContent = lipgloss.JoinVertical(lipgloss.Left, sysGroup, "", resGroup, "", netGroup, "", svcGroup, "", activity)
	} else {
		infoContent = lipgloss.NewStyle().Foreground(dim).Render("\n\nOffline or not connected.\nPress Enter in main view to connect.")
	}

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(dim).
		Padding(1, 3).
		Width(45).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(accent).Bold(true).Render("SERVER TELEMETRY"),
			"",
			infoContent,
		))

	// Combine components
	centerCore := lipgloss.NewStyle().Padding(2, 2, 2, 0).Render(coreArt)
	
	content := lipgloss.JoinHorizontal(lipgloss.Center, leftPanel, centerCore, rightPanel)

	// Header and Footer for NOC view
	header := lipgloss.NewStyle().Foreground(text).Bold(true).Background(lipgloss.Color("236")).Padding(0, 2).Render(" MISSION CONTROL CORE ")
	footer := lipgloss.NewStyle().Foreground(dim).Render("Press [G] or [ESC] to close  |  [Up/Down] Navigate")

	layout := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"\n",
		content,
		"\n\n",
		footer,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, layout)
}
