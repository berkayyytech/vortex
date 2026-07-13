package uptime

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	"main/internal/config"
	uptimeEngine "main/internal/engine/uptime"
	"main/internal/pages"
	"main/internal/theme"
)

type Model struct {
	width  int
	height int
}

func init() {
	pages.Register(New())
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	for _, target := range config.CurrentConfig.UptimeTargets {
		tType := uptimeEngine.HTTPCheck
		if target.Type == "ping" {
			tType = uptimeEngine.PingCheck
		}
		uptimeEngine.AddTarget(&uptimeEngine.MonitorTarget{
			ID:       target.ID,
			Name:     target.Name,
			URL:      target.URL,
			Type:     tType,
			Interval: time.Duration(target.IntervalSecs) * time.Second,
		})
	}
	return tickCmd()
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		// Check for events
		select {
		case alertMsg := <-uptimeEngine.EventChannel:
			return m, tea.Batch(
				tickCmd(),
				func() tea.Msg {
					return pages.LogActivityMsg{Message: alertMsg}
				},
			)
		default:
			return m, tickCmd()
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing Uptime Module..."
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		PaddingBottom(1)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Dim).
		Padding(1, 2)
		
	statusUpStyle := lipgloss.NewStyle().Foreground(theme.Current.Success).Bold(true)
	statusDownStyle := lipgloss.NewStyle().Foreground(theme.Current.Error).Bold(true)

	cardWidth := (m.width - 10) / 3
	if cardWidth < 30 {
		cardWidth = 30
	}

	var cards []string

	for _, t := range uptimeEngine.Targets {
		statusText := t.Status
		if t.Status == "up" {
			statusText = statusUpStyle.Render("UP")
		} else if t.Status == "down" {
			statusText = statusDownStyle.Render("DOWN")
		}
		
		uptimeStr := fmt.Sprintf("%.2f%%", t.Uptime)
		
		var lastResp float64
		if len(t.History) > 0 {
			lastResp = t.History[len(t.History)-1]
		}
		respStr := fmt.Sprintf("%.0fms", lastResp)
		
		color := theme.Current.Success
		if t.Status == "down" {
			color = theme.Current.Error
		}

		cardContent := fmt.Sprintf("%s\n\n%s: %s\n%s: %s\n%s: %s\n\n%s",
			lipgloss.NewStyle().Bold(true).Render(t.Name),
			"Status", statusText,
			"Uptime", lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(uptimeStr),
			"Resp", lipgloss.NewStyle().Foreground(theme.Current.Warning).Render(respStr),
			components.Sparkline(t.History, cardWidth-6, color),
		)

		cards = append(cards, boxStyle.Copy().Width(cardWidth).Render(cardContent))
	}

	var rows []string
	var currentRow []string
	
	for i, card := range cards {
		currentRow = append(currentRow, card)
		if (i+1)%3 == 0 || i == len(cards)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = nil
		}
	}
	
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render("📈 External Uptime Monitor"),
		content,
	)
}

func (m Model) Title() string {
	return "Uptime Monitor"
}

func (m Model) Icon() string {
	return "📈"
}
