package secrets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	secretsengine "main/internal/engine/secrets"
	"main/internal/pages"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

func init() {
	pages.Register(New())
}

type AppState int

const (
	StateBrowsing AppState = iota
	StateViewingVars
	StateEditingVar
	StateDiff
)

type EnvVar struct {
	IsVar    bool
	Raw      string
	Key      string
	Value    string
	Original string
	IsDirty  bool
}

type Model struct {
	engine     *secretsengine.Engine
	state      AppState
	loading    bool
	errMessage string

	files      []string
	fileCursor int

	activeFile string
	vars       []EnvVar
	varCursor  int

	textInput textinput.Model
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 1024
	ti.Width = 60

	return Model{
		state:     StateBrowsing,
		textInput: ti,
		loading:   false,
	}
}

func (m Model) Title() string {
	return "Secrets"
}

func (m Model) Icon() string {
	return "🔒"
}

func (m Model) Init() tea.Cmd {
	return nil
}

type filesResponseMsg struct {
	files []string
	err   error
}

func fetchFilesCmd(engine *secretsengine.Engine) tea.Cmd {
	return func() tea.Msg {
		files, err := engine.SearchEnvFiles()
		return filesResponseMsg{files: files, err: err}
	}
}

type fileReadMsg struct {
	content string
	err     error
}

func readFileCmd(engine *secretsengine.Engine, path string) tea.Cmd {
	return func() tea.Msg {
		content, err := engine.ReadFile(path)
		return fileReadMsg{content: content, err: err}
	}
}

type fileSaveMsg struct {
	err error
}

func saveFileCmd(engine *secretsengine.Engine, path, content string) tea.Cmd {
	return func() tea.Msg {
		err := engine.WriteFile(path, content)
		return fileSaveMsg{err: err}
	}
}

type dbSyncMsg struct {
	err error
}

func syncDbCmd(engine *secretsengine.Engine, path string) tea.Cmd {
	return func() tea.Msg {
		err := engine.SyncToDatabaseManager(path)
		return dbSyncMsg{err: err}
	}
}

func parseEnv(content string) []EnvVar {
	var vars []EnvVar
	lines := strings.Split(content, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			vars = append(vars, EnvVar{IsVar: false, Raw: l})
			continue
		}
		parts := strings.SplitN(l, "=", 2)
		if len(parts) == 2 {
			vars = append(vars, EnvVar{
				IsVar:    true,
				Key:      parts[0],
				Value:    parts[1],
				Original: parts[1],
			})
		} else {
			vars = append(vars, EnvVar{IsVar: false, Raw: l})
		}
	}
	return vars
}

func buildEnv(vars []EnvVar) string {
	var lines []string
	for _, v := range vars {
		if v.IsVar {
			lines = append(lines, fmt.Sprintf("%s=%s", v.Key, v.Value))
		} else {
			lines = append(lines, v.Raw)
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case sshlib.ConnectedMsg:
		m.engine = secretsengine.NewEngine(msg.Client)
		m.loading = true
		return m, fetchFilesCmd(m.engine)

	case filesResponseMsg:
		m.loading = false
		m.files = msg.files
		if msg.err != nil {
			m.errMessage = msg.err.Error()
		} else {
			m.errMessage = ""
		}
		return m, nil

	case fileReadMsg:
		m.loading = false
		if msg.err != nil {
			m.errMessage = msg.err.Error()
			m.state = StateBrowsing
		} else {
			m.vars = parseEnv(msg.content)
			m.state = StateViewingVars
			m.varCursor = 0
			m.errMessage = ""
		}
		return m, nil

	case fileSaveMsg:
		m.loading = false
		if msg.err != nil {
			m.errMessage = "Save Error: " + msg.err.Error()
		} else {
			m.errMessage = "Saved successfully."
			// reset original states
			for i := range m.vars {
				m.vars[i].Original = m.vars[i].Value
				m.vars[i].IsDirty = false
			}
			m.state = StateViewingVars
		}
		return m, nil

	case dbSyncMsg:
		if msg.err != nil {
			m.errMessage = "DB Sync Error: " + msg.err.Error()
		} else {
			m.errMessage = "Successfully integrated with Database Manager."
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch m.state {
		case StateBrowsing:
			switch msg.String() {
			case "up", "k":
				if m.fileCursor > 0 {
					m.fileCursor--
				}
			case "down", "j":
				if m.fileCursor < len(m.files)-1 {
					m.fileCursor++
				}
			case "enter":
				if len(m.files) > 0 {
					m.activeFile = m.files[m.fileCursor]
					m.loading = true
					return m, readFileCmd(m.engine, m.activeFile)
				}
			}

		case StateViewingVars:
			switch msg.String() {
			case "esc":
				m.state = StateBrowsing
			case "up", "k":
				if m.varCursor > 0 {
					m.varCursor--
				}
			case "down", "j":
				if m.varCursor < len(m.vars)-1 {
					m.varCursor++
				}
			case "enter":
				if len(m.vars) > 0 && m.vars[m.varCursor].IsVar {
					m.state = StateEditingVar
					m.textInput.SetValue(m.vars[m.varCursor].Value)
					m.textInput.Focus()
				}
			case "s":
				m.state = StateDiff
			case "d":
				if m.activeFile != "" {
					return m, syncDbCmd(m.engine, m.activeFile)
				}
			}

		case StateEditingVar:
			switch msg.String() {
			case "esc":
				m.state = StateViewingVars
				m.textInput.Blur()
			case "enter":
				m.vars[m.varCursor].Value = m.textInput.Value()
				m.vars[m.varCursor].IsDirty = (m.vars[m.varCursor].Value != m.vars[m.varCursor].Original)
				m.state = StateViewingVars
				m.textInput.Blur()
				
				// Return log message for audit, keeping plaintext secret out of the log
				return m, func() tea.Msg {
					return pages.LogActivityMsg{Message: fmt.Sprintf("Updated %s in %s", m.vars[m.varCursor].Key, m.activeFile)}
				}
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}

		case StateDiff:
			switch msg.String() {
			case "esc", "n":
				m.state = StateViewingVars
			case "y", "enter":
				m.loading = true
				content := buildEnv(m.vars)
				return m, saveFileCmd(m.engine, m.activeFile, content)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(components.Title("SECRETS MANAGER") + "\n\n")

	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("Loading...") + "\n")
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	if m.errMessage != "" {
		if strings.Contains(m.errMessage, "successfully") || strings.Contains(m.errMessage, "Successfully") {
			b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Success).Render(m.errMessage) + "\n\n")
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Error).Render(m.errMessage) + "\n\n")
		}
	}

	switch m.state {
	case StateBrowsing:
		b.WriteString("Select a .env file:\n\n")
		if len(m.files) == 0 {
			b.WriteString("No .env files found.")
		} else {
			for i, f := range m.files {
				cursor := "  "
				if m.fileCursor == i {
					cursor = "> "
					b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(cursor + f) + "\n")
				} else {
					b.WriteString(cursor + f + "\n")
				}
			}
		}

	case StateViewingVars:
		b.WriteString(fmt.Sprintf("Editing %s \n[s]ave  [d]b-sync  [esc]back  [enter]edit\n\n", m.activeFile))
		for i, v := range m.vars {
			if !v.IsVar {
				b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("  " + v.Raw) + "\n")
				continue
			}

			cursor := "  "
			if m.varCursor == i {
				cursor = "> "
			}

			valDisplay := "********"
			line := fmt.Sprintf("%s%s=%s", cursor, v.Key, valDisplay)
			
			if m.varCursor == i {
				b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Primary).Render(line) + "\n")
			} else {
				if v.IsDirty {
					b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Warning).Render(line) + "\n")
				} else {
					b.WriteString(line + "\n")
				}
			}
		}

	case StateEditingVar:
		b.WriteString(fmt.Sprintf("Editing %s\n\n", m.activeFile))
		b.WriteString(fmt.Sprintf("Key: %s\n", m.vars[m.varCursor].Key))
		b.WriteString(m.textInput.View() + "\n\n")
		b.WriteString("[enter] apply   [esc] cancel\n")

	case StateDiff:
		b.WriteString("Diff View - Confirm Save? [y/N]\n\n")
		hasChanges := false
		for _, v := range m.vars {
			if v.IsVar && v.IsDirty {
				hasChanges = true
				b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Error).Render(fmt.Sprintf("- %s=********", v.Key)) + "\n")
				b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Success).Render(fmt.Sprintf("+ %s=********", v.Key)) + "\n")
			}
		}
		if !hasChanges {
			b.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("No changes detected."))
		}
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}
