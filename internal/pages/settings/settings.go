package settings

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/theme"
)

type Model struct {
	cursor int
}

func New() Model {
	return Model{cursor: 0}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(theme.Themes)-1 {
				m.cursor++
			}
		case "enter":
			// Apply the selected theme!
			theme.Current = theme.Themes[m.cursor]
		}
	}
	return m, nil
}

func (m Model) View() string {
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Dim).
		Padding(1, 3).
		Margin(1, 0)

	titleCard := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Accent).
		MarginBottom(1)

	var items string
	for i, t := range theme.Themes {
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(theme.Current.Text)
		if m.cursor == i {
			cursor = "▶ "
			style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
		}
		
		// Add an indicator for the currently active theme
		activeTag := ""
		if t.Name == theme.Current.Name {
			activeTag = " (Active)"
		}

		items += fmt.Sprintf("%s %s%s\n", cursor, style.Render(t.Name), activeTag)
	}

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("UI THEME ENGINE"),
			items,
			"",
			lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("Select a theme and press ENTER to apply globally instantly."),
		),
	)
}

func (m Model) Title() string { return "Settings" }
func (m Model) Icon() string { return "🛠️" }
