package proxy

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/agent"
	"main/internal/components"
	proxyengine "main/internal/engine/proxy"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type AppState int

const (
	StateBrowsing AppState = iota
	StateEditing
)

type Model struct {
	engine       *proxyengine.Engine
	state        AppState
	proxyType    proxyengine.ProxyType
	sites        []proxyengine.Site
	cursor       int
	
	textArea     textarea.Model
	editingFile  string
	isDirty      bool
	
	loading      bool
	errorMessage string
	
	valOutput    string
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Empty configuration..."
	ta.SetWidth(80)
	ta.SetHeight(20)

	return Model{
		state:     StateBrowsing,
		textArea:  ta,
		loading:   true,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

type detectResultMsg struct {
	ptype proxyengine.ProxyType
	sites []proxyengine.Site
	err   error
}

func detectProxy(engine *proxyengine.Engine) tea.Cmd {
	return func() tea.Msg {
		if engine == nil {
			return nil
		}
		ptype := engine.DetectProxy()
		if ptype == proxyengine.ProxyNone {
			return detectResultMsg{ptype: ptype, err: fmt.Errorf("no supported reverse proxy detected")}
		}
		sites, err := engine.ListSites(ptype)
		return detectResultMsg{ptype: ptype, sites: sites, err: err}
	}
}

type fileReadMsg struct {
	content string
	err     error
}

func readConfig(engine *proxyengine.Engine, path string) tea.Cmd {
	return func() tea.Msg {
		content, err := engine.ReadConfig(path)
		return fileReadMsg{content: content, err: err}
	}
}

type fileSaveMsg struct {
	err error
}

func saveConfig(engine *proxyengine.Engine, path, content string) tea.Cmd {
	return func() tea.Msg {
		err := engine.WriteConfig(path, content)
		return fileSaveMsg{err: err}
	}
}

type validationResultMsg struct {
	output string
	err    error
}

func validateAndReload(engine *proxyengine.Engine, ptype proxyengine.ProxyType) tea.Cmd {
	return func() tea.Msg {
		out, err := engine.ValidateSyntax(ptype)
		if err != nil {
			return validationResultMsg{output: out, err: fmt.Errorf("validation failed")}
		}
		
		rOut, rErr := engine.ReloadRestart(ptype)
		if rErr != nil {
			return validationResultMsg{output: out + "\n" + rOut, err: fmt.Errorf("reload failed")}
		}
		
		return validationResultMsg{output: out + "\n" + rOut + "\nReload successful.", err: nil}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case detectResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = "Error: " + msg.err.Error()
			m.sites = []proxyengine.Site{}
		} else {
			m.errorMessage = ""
			m.proxyType = msg.ptype
			m.sites = msg.sites
			if m.cursor >= len(m.sites) {
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
			m.errorMessage = "File saved successfully. Press CTRL+V to validate & reload."
		}
		return m, nil

	case validationResultMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = "Validation Error: " + msg.err.Error()
		} else {
			m.errorMessage = "Config Validated & Reloaded!"
		}
		m.valOutput = msg.output
		return m, nil

	case sshlib.ConnectedMsg:
		m.engine = proxyengine.NewEngine(msg.Client)
		m.loading = true
		return m, detectProxy(m.engine)

	case agent.Payload:
		if m.engine != nil && m.state == StateBrowsing && m.proxyType == "" {
			m.loading = true
			return m, detectProxy(m.engine)
		}
	}

	switch m.state {
	case StateEditing:
		var cmd tea.Cmd
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.state = StateBrowsing
				m.errorMessage = ""
				m.valOutput = ""
				return m, nil
			case "ctrl+s":
				return m, saveConfig(m.engine, m.editingFile, m.textArea.Value())
			case "ctrl+v":
				m.loading = true
				return m, validateAndReload(m.engine, m.proxyType)
			default:
				m.textArea, cmd = m.textArea.Update(msg)
				m.isDirty = true
				return m, cmd
			}
		default:
			m.textArea, cmd = m.textArea.Update(msg)
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
				if m.cursor < len(m.sites)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.sites) > 0 {
					selected := m.sites[m.cursor]
					m.state = StateEditing
					m.editingFile = selected.ConfPath
					m.isDirty = false
					m.errorMessage = ""
					m.valOutput = ""
					m.textArea.SetValue("Loading...")
					m.textArea.Focus()
					m.loading = true
					return m, readConfig(m.engine, selected.ConfPath)
				}
			case "r":
				m.loading = true
				return m, detectProxy(m.engine)
			case "v":
				if m.proxyType != proxyengine.ProxyNone {
					m.loading = true
					return m, validateAndReload(m.engine, m.proxyType)
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var content string

	headerStyle := lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
	breadcrumbs := fmt.Sprintf("Proxy Type: %s", m.proxyType)
	if m.proxyType == "" {
		breadcrumbs = "Detecting proxy..."
	}
	
	if m.errorMessage != "" {
		breadcrumbs += lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("   " + m.errorMessage)
	}

	content = lipgloss.JoinVertical(lipgloss.Left,
		components.Title("REVERSE PROXY MANAGER"),
		headerStyle.Render(breadcrumbs)+"\n",
	)

	switch m.state {
	case StateEditing:
		status := " [Saved]"
		if m.isDirty {
			status = " [Unsaved]"
		}
		editorHeader := lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(fmt.Sprintf("Editing: %s %s", m.editingFile, status))
		controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [ESC] Back  [CTRL+S] Save  [CTRL+V] Validate & Reload")
		
		valView := ""
		if m.valOutput != "" {
			valView = lipgloss.NewStyle().Foreground(theme.Current.Warning).Render("\nLive Output:\n" + m.valOutput)
		}

		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			editorHeader,
			"",
			m.textArea.View(),
			valView,
			controls,
		)

	case StateBrowsing:
		if m.loading {
			content = lipgloss.JoinVertical(lipgloss.Left, content, "Loading...")
		} else if len(m.sites) == 0 {
			content = lipgloss.JoinVertical(lipgloss.Left, content, "No sites found.")
		} else {
			colName := 30
			colTarget := 20
			colSSL := 15
			
			headerRow := fmt.Sprintf("  %-*s %-*s %-*s", 
				colName, "Site Name", 
				colTarget, "Target", 
				colSSL, "SSL Status")
				
			items := lipgloss.NewStyle().Foreground(theme.Current.Dim).Bold(true).Render(headerRow) + "\n"

			start := 0
			maxLines := 15
			if m.cursor > maxLines/2 {
				start = m.cursor - maxLines/2
			}
			end := start + maxLines
			if end > len(m.sites) {
				end = len(m.sites)
				start = end - maxLines
				if start < 0 {
					start = 0
				}
			}

			for i := start; i < end; i++ {
				site := m.sites[i]
				cursor := "  "
				style := lipgloss.NewStyle().Foreground(theme.Current.Text)
				
				if m.cursor == i {
					cursor = "▶ "
					style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
				}

				sslStyle := lipgloss.NewStyle().Foreground(theme.Current.Text)
				if site.SSLStatus == "Active" || site.SSLStatus == "Auto" {
					sslStyle = lipgloss.NewStyle().Foreground(theme.Current.Success)
				} else {
					sslStyle = lipgloss.NewStyle().Foreground(theme.Current.Dim)
				}

				displayNm := site.Name
				if len(displayNm) > colName-3 {
					displayNm = displayNm[:colName-3] + "..."
				}

				row := fmt.Sprintf("🌐 %-*s %-*s %-*s",
					colName-2, style.Render(displayNm),
					colTarget, site.Target,
					colSSL, sslStyle.Render(site.SSLStatus))

				items += fmt.Sprintf("%s%s\n", cursor, row)
			}
			content = lipgloss.JoinVertical(lipgloss.Left, content, items)
		}

		controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [↑/↓] Navigate  [ENTER] Edit Config  [r] Refresh  [v] Validate & Reload All")
		content = lipgloss.JoinVertical(lipgloss.Left, content, controls)
	}

	return components.Card(content, 90)
}

func (m Model) Title() string { return "Proxy" }
func (m Model) Icon() string { return "🌐" }
