package dashboard

import (
	"fmt"
	"strings"

	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/stats"
	"main/internal/theme"
)

// Model holds the state for the Dashboard page.
type Model struct {
	sysStats stats.SystemStats
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
	case statsMsg:
		m.sysStats = stats.SystemStats(msg)
		return m, nil
	case agent.Payload:
		m.sysStats = msg.Stats
		return m, nil
	case tickMsg:
		return m, tea.Batch(fetchStatsCmd, tickCmd())
	}
	return m, nil
}

func colorizeBar(percent float64) string {
	totalBlocks := 30
	filledBlocks := int((percent / 100.0) * float64(totalBlocks))
	if filledBlocks < 0 { filledBlocks = 0 }
	if filledBlocks > totalBlocks { filledBlocks = totalBlocks }
	
	filled := strings.Repeat("█", filledBlocks)
	empty := strings.Repeat("░", totalBlocks-filledBlocks)

	var color lipgloss.Color
	if percent < 50 {
		color = lipgloss.Color("46") // Green
	} else if percent < 80 {
		color = lipgloss.Color("226") // Yellow
	} else {
		color = lipgloss.Color("196") // Red
	}

	return lipgloss.NewStyle().Foreground(color).Render(filled) + 
	       lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(empty) +
	       lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf(" %3d%%", int(percent)))
}

func (m Model) View() string {
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Dim).
		Padding(1, 3).
		Margin(1, 0).
		Width(60)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Accent).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().Foreground(theme.Current.Text).Width(8)

	return card.Render(
		strings.Join([]string{
			titleCard.Render("HARDWARE UTILIZATION"),
			"",
			labelStyle.Render("CPU") + " " + colorizeBar(m.sysStats.CPUPercent),
			labelStyle.Render("RAM") + " " + colorizeBar(m.sysStats.RAMPercent),
			labelStyle.Render("Disk") + " " + colorizeBar(m.sysStats.DiskPercent),
			"",
			titleCard.Render("SYSTEM INFORMATION"),
			"",
			fmt.Sprintf("%s %s", labelStyle.Render("Uptime"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.Uptime)),
			fmt.Sprintf("%s %s", labelStyle.Render("OS"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.OS)),
			fmt.Sprintf("%s %s", labelStyle.Render("Kernel"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.Kernel)),
		}, "\n"),
	)
}

func (m Model) Title() string {
	return "Dashboard"
}

func (m Model) Icon() string {
	return "📊"
}
