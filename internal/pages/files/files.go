package files

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	fileengine "main/internal/engine/files"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type AppState int

const (
	StateBrowsing AppState = iota
	StatePrompting
	StateEditing
)

type OperationMode string

const (
	ModeNone   OperationMode = ""
	ModeRename OperationMode = "Rename"
	ModeCopy   OperationMode = "Copy"
	ModeMove   OperationMode = "Move"
	ModeChmod  OperationMode = "Chmod"
	ModeDelete OperationMode = "Delete"
)

type Model struct {
	cwd         string
	files       []fileengine.FileInfo
	cursor      int
	engine      *fileengine.Engine
	state       AppState
	
	opMode      OperationMode
	textInput   textinput.Model
	
	textArea    textarea.Model
	editingFile string
	isDirty     bool
	
	errorMessage string
	loading      bool
}

func New() Model {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 40

	ta := textarea.New()
	ta.Placeholder = "Empty file..."
	ta.SetWidth(80)
	ta.SetHeight(20)

	return Model{
		cwd:       "/",
		files:     []fileengine.FileInfo{},
		cursor:    0,
		state:     StateBrowsing,
		textInput: ti,
		textArea:  ta,
		loading:   true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

type filesResponseMsg struct {
	files []fileengine.FileInfo
	err   error
}

func fetchDir(engine *fileengine.Engine, dir string) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		files, err := engine.ListDirectory(dir)
		return filesResponseMsg{files: files, err: err}
	}
}

type fileReadMsg struct {
	content string
	err     error
}

func readFile(engine *fileengine.Engine, path string) tea.Cmd {
	return func() tea.Msg {
		content, err := engine.ReadFile(path)
		return fileReadMsg{content: content, err: err}
	}
}

type fileSaveMsg struct {
	err error
}

func saveFile(engine *fileengine.Engine, path, content string) tea.Cmd {
	return func() tea.Msg {
		err := engine.WriteFile(path, content)
		return fileSaveMsg{err: err}
	}
}

type operationCompleteMsg struct {
	err error
}

func doOperation(engine *fileengine.Engine, op OperationMode, cwd, target, arg string) tea.Cmd {
	return func() tea.Msg {
		var err error
		targetPath := path.Join(cwd, target)
		switch op {
		case ModeRename:
			err = engine.Rename(targetPath, path.Join(cwd, arg))
		case ModeCopy:
			err = engine.Copy(targetPath, path.Join(cwd, arg))
		case ModeMove:
			err = engine.Move(targetPath, path.Join(cwd, arg))
		case ModeChmod:
			err = engine.Chmod(targetPath, arg)
		case ModeDelete:
			err = engine.Delete(targetPath)
		}
		return operationCompleteMsg{err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case filesResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = "Error: " + msg.err.Error()
			m.files = []fileengine.FileInfo{}
		} else {
			m.errorMessage = ""
			m.files = msg.files
			if m.cursor >= len(m.files) {
				m.cursor = 0
			}
		}
		return m, nil

	case fileReadMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = "Error reading file: " + msg.err.Error()
			m.state = StateBrowsing
		} else {
			m.textArea.SetValue(msg.content)
			m.isDirty = false
		}
		return m, nil

	case fileSaveMsg:
		if msg.err != nil {
			m.errorMessage = "Error saving file: " + msg.err.Error()
		} else {
			m.isDirty = false
			m.errorMessage = "File saved successfully."
		}
		return m, nil

	case operationCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = "Operation failed: " + msg.err.Error()
		} else {
			m.errorMessage = ""
		}
		m.state = StateBrowsing
		m.loading = true
		return m, fetchDir(m.engine, m.cwd)

	case sshlib.ConnectedMsg:
		m.engine = fileengine.NewEngine(msg.Client)
		m.loading = true
		return m, fetchDir(m.engine, m.cwd)

	case agent.Payload:
		if m.engine != nil && m.state == StateBrowsing {
			m.loading = true
			return m, fetchDir(m.engine, m.cwd)
		}
	}

	// State specific input handling
	switch m.state {
	case StateEditing:
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.state = StateBrowsing
				m.errorMessage = ""
				return m, nil
			case "ctrl+s":
				return m, saveFile(m.engine, path.Join(m.cwd, m.editingFile), m.textArea.Value())
			default:
				m.textArea, cmd = m.textArea.Update(msg)
				m.isDirty = true
				return m, cmd
			}
		default:
			m.textArea, cmd = m.textArea.Update(msg)
			return m, cmd
		}

	case StatePrompting:
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.state = StateBrowsing
				m.errorMessage = ""
				return m, nil
			case "enter":
				val := m.textInput.Value()
				if m.opMode == ModeDelete && strings.ToLower(val) != "y" {
					m.state = StateBrowsing
					m.errorMessage = "Delete cancelled."
					return m, nil
				}
				
				target := m.files[m.cursor].Name
				if m.opMode == ModeDelete {
					target = strings.TrimSuffix(target, "/")
				}
				
				m.loading = true
				return m, doOperation(m.engine, m.opMode, m.cwd, m.files[m.cursor].Name, val)
			default:
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		default:
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

	case StateBrowsing:
		switch msg := msg.(type) {
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
					if selected.IsDir {
						m.cwd = path.Join(m.cwd, selected.Name)
						m.cursor = 0
						m.loading = true
						return m, fetchDir(m.engine, m.cwd)
					} else {
						// Open Editor
						m.state = StateEditing
						m.editingFile = selected.Name
						m.isDirty = false
						m.errorMessage = ""
						m.textArea.SetValue("Loading...")
						m.textArea.Focus()
						m.loading = true
						return m, readFile(m.engine, path.Join(m.cwd, selected.Name))
					}
				}
			case "backspace", "b", "h":
				m.cwd = path.Dir(strings.TrimSuffix(m.cwd, "/"))
				if m.cwd == "" {
					m.cwd = "/"
				}
				m.cursor = 0
				m.loading = true
				return m, fetchDir(m.engine, m.cwd)
				
			case "r": // Rename
				if len(m.files) > 0 {
					m.state = StatePrompting
					m.opMode = ModeRename
					m.textInput.Placeholder = "New name"
					m.textInput.SetValue(m.files[m.cursor].Name)
					m.textInput.Focus()
				}
			case "c": // Copy
				if len(m.files) > 0 {
					m.state = StatePrompting
					m.opMode = ModeCopy
					m.textInput.Placeholder = "Destination path"
					m.textInput.SetValue(m.files[m.cursor].Name + "_copy")
					m.textInput.Focus()
				}
			case "m": // Move
				if len(m.files) > 0 {
					m.state = StatePrompting
					m.opMode = ModeMove
					m.textInput.Placeholder = "Destination path"
					m.textInput.SetValue(m.files[m.cursor].Name)
					m.textInput.Focus()
				}
			case "x": // Chmod
				if len(m.files) > 0 {
					m.state = StatePrompting
					m.opMode = ModeChmod
					m.textInput.Placeholder = "e.g. 755"
					m.textInput.SetValue("")
					m.textInput.Focus()
				}
			case "d": // Delete
				if len(m.files) > 0 {
					m.state = StatePrompting
					m.opMode = ModeDelete
					m.textInput.Placeholder = "Type 'y' to confirm"
					m.textInput.SetValue("")
					m.textInput.Focus()
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var content string

	// Common header with breadcrumbs
	headerStyle := lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
	breadcrumbs := fmt.Sprintf("CWD: %s", m.cwd)
	
	if m.errorMessage != "" {
		breadcrumbs += lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("   " + m.errorMessage)
	}
	
	content = lipgloss.JoinVertical(lipgloss.Left,
		components.Title("REMOTE FILE EXPLORER"),
		headerStyle.Render(breadcrumbs)+"\n",
	)

	switch m.state {
	case StateEditing:
		status := " [Saved]"
		if m.isDirty {
			status = " [Unsaved]"
		}
		editorHeader := lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(fmt.Sprintf("Editing: %s %s", m.editingFile, status))
		controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [ESC] Back  [CTRL+S] Save")
		
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			editorHeader,
			"",
			m.textArea.View(),
			controls,
		)

	case StatePrompting:
		promptLabel := ""
		switch m.opMode {
		case ModeRename:
			promptLabel = "Rename " + m.files[m.cursor].Name + " to:"
		case ModeCopy:
			promptLabel = "Copy " + m.files[m.cursor].Name + " to:"
		case ModeMove:
			promptLabel = "Move " + m.files[m.cursor].Name + " to:"
		case ModeChmod:
			promptLabel = "Change mode for " + m.files[m.cursor].Name + ":"
		case ModeDelete:
			promptLabel = "Are you sure you want to delete " + m.files[m.cursor].Name + "?"
		}
		
		controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [ENTER] Submit  [ESC] Cancel")
		
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(promptLabel),
			m.textInput.View(),
			controls,
		)

	case StateBrowsing:
		if m.loading {
			content = lipgloss.JoinVertical(lipgloss.Left, content, "Loading...")
		} else if len(m.files) == 0 {
			content = lipgloss.JoinVertical(lipgloss.Left, content, "Directory is empty.")
		} else {
			// Fixed columns
			colName := 35
			colSize := 10
			colPerm := 12
			colOwner := 15
			colMod := 20
			
			headerRow := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s", 
				colName, "Name", 
				colSize, "Size", 
				colPerm, "Perms", 
				colOwner, "Owner", 
				colMod, "Modified")
				
			items := lipgloss.NewStyle().Foreground(theme.Current.Dim).Bold(true).Render(headerRow) + "\n"

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
				cursor := "  "
				style := lipgloss.NewStyle().Foreground(theme.Current.Text)
				
				if m.cursor == i {
					cursor = "▶ "
					style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
				}

				icon := "📄"
				if f.IsDir {
					icon = "📁"
					style = style.Foreground(theme.Current.Accent)
				}
				
				// Semantic color for permission risk
				permStyle := lipgloss.NewStyle().Foreground(theme.Current.Text)
				if len(f.Permissions) > 8 && f.Permissions[8] == 'w' {
					permStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")) // Red for world-writable
				} else if len(f.Permissions) > 5 && f.Permissions[5] == 'w' {
					permStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")) // Yellow for group-writable
				}
				
				// Truncate name if too long
				displayNm := f.Name
				if len(displayNm) > colName-3 {
					displayNm = displayNm[:colName-6] + "..."
				}

				row := fmt.Sprintf("%s %-*s %-*s %s %-*s %-*s",
					icon,
					colName-2, style.Render(displayNm),
					colSize, f.Size,
					permStyle.Render(fmt.Sprintf("%-*s", colPerm, f.Permissions)),
					colOwner, f.Owner,
					colMod, f.Modified)

				items += fmt.Sprintf("%s%s\n", cursor, row)
			}
			content = lipgloss.JoinVertical(lipgloss.Left, content, items)
		}

		controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [↑/↓] Navigate  [ENTER] Open  [BACKSPACE] Go Up\nOperations: [r]ename  [c]opy  [m]ove  [x]chmod  [d]elete")
		content = lipgloss.JoinVertical(lipgloss.Left, content, controls)
	}

	return components.Card(content, 90)
}

func (m Model) Title() string { return "Files" }
func (m Model) Icon() string { return "📁" }
