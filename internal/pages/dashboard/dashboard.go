package dashboard

import (
	"fmt"
	"strings"

	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/stats"
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
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	return card.Render(
		strings.Join([]string{
			titleCard.Render("HARDWARE UTILIZATION"),
			fmt.Sprintf("CPU      %s", stats.FormatBar(m.sysStats.CPUPercent)),
			fmt.Sprintf("RAM      %s", stats.FormatBar(m.sysStats.RAMPercent)),
			fmt.Sprintf("Disk     %s", stats.FormatBar(m.sysStats.DiskPercent)),
			"",
			fmt.Sprintf("Uptime   %s", m.sysStats.Uptime),
			fmt.Sprintf("OS       %s", m.sysStats.OS),
			fmt.Sprintf("Kernel   %s", m.sysStats.Kernel),
		}, "\n"),
	)
}

func (m Model) Title() string {
	return "Dashboard"
}

func (m Model) Icon() string {
	return "📊"
}
