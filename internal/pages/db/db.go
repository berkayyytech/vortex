package db

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	dbengine "main/internal/engine/db"
	"main/internal/pages"
	"main/internal/theme"
)

type fetchedStatsMsg dbengine.DatabaseStats
type queryResultMsg string

type Model struct {
	engine      *dbengine.Engine
	stats       dbengine.DatabaseStats
	isFetching  bool

	// Query Console state
	queryInput  string
	queryResult string
	cursor      int // 0 = Postges, 1 = MySQL, 2 = Redis
	inputActive bool
}

func init() {
	pages.Register(New())
}

func New() Model {
	return Model{
		cursor: 0,
		stats: dbengine.DatabaseStats{
			PostgresStatus: "Unknown",
			MySQLStatus:    "Unknown",
			RedisStatus:    "Unknown",
		},
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Title() string { return "Databases" }
func (m Model) Icon() string { return "🗄" }

func (m Model) IsInputActive() bool {
	return m.inputActive
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case pages.EngineReadyMsg:
		m.engine = dbengine.NewEngine(msg.Client)
		if !m.isFetching {
			m.isFetching = true
			cmds = append(cmds, func() tea.Msg {
				return fetchedStatsMsg(m.engine.FetchStats())
			})
		}

	case fetchedStatsMsg:
		m.isFetching = false
		m.stats = dbengine.DatabaseStats(msg)

	case queryResultMsg:
		m.queryResult = string(msg)
		m.inputActive = false

	case tea.KeyMsg:
		if m.inputActive {
			switch msg.Type {
			case tea.KeyEsc:
				m.inputActive = false
			case tea.KeyEnter:
				if m.engine != nil && m.queryInput != "" {
					dbType := "postgres"
					if m.cursor == 1 {
						dbType = "mysql"
					}
					if m.cursor == 2 {
						dbType = "redis"
					}

					query := m.queryInput
					cmds = append(cmds, func() tea.Msg {
						return queryResultMsg(m.engine.RunQuery(dbType, query))
					})
					m.queryInput = "" // clear for next time, but stay active until query returns? Or just turn off.
					// we turn it off on response
				} else {
					m.inputActive = false
				}
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.queryInput) > 0 {
					m.queryInput = m.queryInput[:len(m.queryInput)-1]
				}
			case tea.KeySpace:
				m.queryInput += " "
			case tea.KeyRunes:
				m.queryInput += string(msg.Runes)
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 2 {
				m.cursor++
			}
		case "r", "R":
			if m.engine != nil && !m.isFetching {
				m.isFetching = true
				cmds = append(cmds, func() tea.Msg {
					return fetchedStatsMsg(m.engine.FetchStats())
				})
			}
		case "enter":
			m.inputActive = true
			m.queryInput = ""
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	accent := theme.Current.Accent
	primary := theme.Current.Primary
	text := theme.Current.Text
	dim := theme.Current.Dim

	header := fmt.Sprintf("Databases Managed: 3 | Last Refresh: Just now")

	dbs := []struct {
		Name   string
		Status string
		Conns  string
		Size   string
	}{
		{"PostgreSQL", m.stats.PostgresStatus, m.stats.PostgresActiveConns, m.stats.PostgresSize},
		{"MySQL", m.stats.MySQLStatus, m.stats.MySQLActiveConns, m.stats.MySQLSize},
		{"Redis", m.stats.RedisStatus, m.stats.RedisActiveConns, m.stats.RedisSize},
	}

	var list string
	for i, db := range dbs {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(text)
		if m.cursor == i {
			cursor = "▶ "
			style = lipgloss.NewStyle().Foreground(primary).Bold(true)
		}

		statusColor := dim
		if db.Status == "active" {
			statusColor = lipgloss.Color("46")
		} else if db.Status != "" && db.Status != "Unknown" {
			statusColor = lipgloss.Color("196")
		}

		statusStr := lipgloss.NewStyle().Foreground(statusColor).Render(fmt.Sprintf("[%s]", db.Status))

		details := fmt.Sprintf("Conns: %s | Size: %s", db.Conns, db.Size)
		list += fmt.Sprintf("%s %s %s - %s\n", cursor, statusStr, style.Render(db.Name), lipgloss.NewStyle().Foreground(dim).Render(details))
	}

	controls := lipgloss.NewStyle().Foreground(dim).Render("\nControls: [up/down] Select DB  [Enter] Open Query Console  [R] Refresh")

	consoleView := ""
	if m.inputActive {
		consoleView = "\n\n" + lipgloss.NewStyle().Foreground(accent).Bold(true).Render("Query Console (Type and press Enter): ") + "\n" + m.queryInput + "█"
	}

	resView := ""
	if m.queryResult != "" {
		// Just a bit of formatting
		formattedRes := strings.TrimSpace(m.queryResult)
		if len(formattedRes) > 500 {
			formattedRes = formattedRes[:500] + "\n... (truncated)"
		}
		resView = "\n\n" + lipgloss.NewStyle().Foreground(primary).Bold(true).Render("Query Result:") + "\n" + lipgloss.NewStyle().Foreground(text).Render(formattedRes)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("DATABASE MANAGER v1.9"),
		header,
		"\n"+list,
		controls,
		consoleView,
		resView,
	)

	return components.Card(content, 80)
}
