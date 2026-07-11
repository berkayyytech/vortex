package files

import (
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/pages"
	"main/internal/theme"
)

type Model struct {
	cwd    string
	files  []string
	cursor int
}

func New() Model {
	return Model{
		cwd:    "/",
		files:  []string{"Loading..."},
		cursor: 0,
	}
}

func (m Model) Init() tea.Cmd { return nil }

type filesResponseMsg struct {
	files []string
}

func fetchDir(dir string) tea.Cmd {
	return func() tea.Msg {
		return pages.RunRemoteQueryMsg{
			Command: "ls -1pA " + dir,
			ResponseHandler: func(out string) tea.Msg {
				lines := strings.Split(strings.TrimSpace(out), "\n")
				return filesResponseMsg{files: lines}
			},
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case filesResponseMsg:
		m.files = msg.files
		if m.cursor >= len(m.files) {
			m.cursor = 0
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.files) > 0 {
				selected := m.files[m.cursor]
				if strings.HasSuffix(selected, "/") {
					m.cwd = path.Join(m.cwd, selected)
					m.cursor = 0
					m.files = []string{"Loading..."}
					return m, fetchDir(m.cwd)
				}
			}
		case "backspace", "b":
			m.cwd = path.Dir(m.cwd)
			m.cursor = 0
			m.files = []string{"Loading..."}
			return m, fetchDir(m.cwd)
		}

	case agent.Payload:
		// Automatically refresh current directory every 5s on payload tick
		return m, fetchDir(m.cwd)
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
	for i, f := range m.files {
		if f == "" {
			continue
		}
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(theme.Current.Text)
		
		if m.cursor == i {
			cursor = "▶ "
			style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
		}

		icon := "📄"
		if strings.HasSuffix(f, "/") {
			icon = "📁"
			style = style.Foreground(theme.Current.Accent)
		}

		items += fmt.Sprintf("%s %s %s\n", cursor, icon, style.Render(f))
	}

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [up/down] Navigate  [ENTER] Enter Directory  [BACKSPACE] Go Up")

	return card.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			titleCard.Render("REMOTE FILE EXPLORER"),
			lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("CWD: "+m.cwd)+"\n",
			items,
			controls,
		),
	)
}

func (m Model) Title() string { return "Files" }
func (m Model) Icon() string { return "📁" }
