package dashboard

import (
	"fmt"
	"strings"

	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
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

func (m Model) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(theme.Current.Text).Width(8)

	content := strings.Join([]string{
		components.Title("HARDWARE UTILIZATION"),
		"",
		labelStyle.Render("CPU") + " " + components.ProgressBar(m.sysStats.CPUPercent, 30),
		labelStyle.Render("RAM") + " " + components.ProgressBar(m.sysStats.RAMPercent, 30),
		labelStyle.Render("Disk") + " " + components.ProgressBar(m.sysStats.DiskPercent, 30),
		"",
		components.Title("SYSTEM INFORMATION"),
		"",
		fmt.Sprintf("%s %s", labelStyle.Render("Uptime"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.Uptime)),
		fmt.Sprintf("%s %s", labelStyle.Render("OS"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.OS)),
		fmt.Sprintf("%s %s", labelStyle.Render("Kernel"), lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(m.sysStats.Kernel)),
	}, "\n")

	return components.Card(content, 60)
}

func (m Model) Title() string {
	return "Dashboard"
}

func (m Model) Icon() string {
	return "📊"
}
