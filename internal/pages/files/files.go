package files

import (
	"fmt"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	fileengine "main/internal/engine/files"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Model struct {
	cwd    string
	files  []string
	cursor int
	engine *fileengine.Engine
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

func fetchDir(engine *fileengine.Engine, dir string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		files, err := engine.ListDirectory(dir)
		if err != nil {
			return filesResponseMsg{files: []string{"Error: " + err.Error()}}
		}
		return filesResponseMsg{files: files}
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
					return m, fetchDir(m.engine, m.cwd)
				}
			}
		case "backspace", "b":
			m.cwd = path.Dir(m.cwd)
			m.cursor = 0
			m.files = []string{"Loading..."}
			return m, fetchDir(m.engine, m.cwd)
		}

	case sshlib.ConnectedMsg:
		m.engine = fileengine.NewEngine(msg.Client)
		return m, fetchDir(m.engine, m.cwd)

	case agent.Payload:
		// Refresh when payload arrives if we have an engine
		return m, fetchDir(m.engine, m.cwd)
	}
	return m, nil
}

func (m Model) View() string {
	var items string
	
	start := 0
	maxLines := 15
	if m.cursor > maxLines/2 {
		start = m.cursor - maxLines/2
	}
	end := start + maxLines
	if end > len(m.files) {
		end = len(m.files)
		start = end - maxLines
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		f := m.files[i]
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

	content := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("REMOTE FILE EXPLORER"),
		lipgloss.NewStyle().Foreground(theme.Current.Primary).Render("CWD: "+m.cwd)+"\n",
		items,
		controls,
	)

	return components.Card(content, 60)
}

func (m Model) Title() string { return "Files" }
func (m Model) Icon() string { return "📁" }
