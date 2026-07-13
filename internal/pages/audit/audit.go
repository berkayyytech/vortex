package audit

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	auditengine "main/internal/engine/audit"
	"main/internal/pages"
	"main/internal/theme"
)

type Model struct {
	searchInput textinput.Model
	isSearching bool
	cursor      int
	logs        []auditengine.LogEntry
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Search logs..."
	ti.CharLimit = 156
	ti.Width = 50

	return Model{
		searchInput: ti,
		logs:        auditengine.GlobalEngine.GetAll(),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isSearching {
			switch msg.String() {
			case "enter", "esc":
				m.isSearching = false
				m.searchInput.Blur()
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.logs = auditengine.GlobalEngine.Search(m.searchInput.Value())
				m.cursor = 0
				return m, cmd
			}
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.logs)-1 {
				m.cursor++
			}
		case "/":
			m.isSearching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		}
		
	case pages.LogActivityMsg:
		// Reload logs
		if m.searchInput.Value() == "" {
			m.logs = auditengine.GlobalEngine.GetAll()
		} else {
			m.logs = auditengine.GlobalEngine.Search(m.searchInput.Value())
		}
	}

	return m, nil
}

func (m Model) IsInputActive() bool {
	return m.isSearching
}

func (m Model) View() string {
	header := "Press '/' to search, 'Esc' or 'Enter' to exit search mode.\n\n"
	header += m.searchInput.View() + "\n\n"

	var list string
	if len(m.logs) == 0 {
		list = "No logs found."
	} else {
		for i, l := range m.logs {
			if i < m.cursor-5 || i > m.cursor+15 {
				continue // simple pagination/window
			}
			
			cursor := "  "
			style := lipgloss.NewStyle().Foreground(theme.Current.Text)
			
			if m.cursor == i {
				cursor = "▶ "
				style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			}
			
			timeStr := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render(l.Timestamp.Format("2006-01-02 15:04:05"))
			list += fmt.Sprintf("%s %s  %s\n", cursor, timeStr, style.Render(l.Message))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("AUDIT LOG"),
		header,
		list,
	)

	return components.Card(content, 80)
}

func (m Model) Title() string { return "Audit Log" }
func (m Model) Icon() string { return "📋" }
