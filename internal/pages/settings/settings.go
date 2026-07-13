package settings

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/components"
	"main/internal/config"
	"main/internal/theme"
)

type Model struct {
	categories    []string
	activeCatIdx  int
	activeItemIdx int
	focusedPane   int // 0 = Sidebar, 1 = Items
	statusMsg     string

	// Input modes
	capturingMode string // "none", "text", "key"
	input         textinput.Model
}

func New() Model {
	catMap := make(map[string]bool)
	var cats []string
	for _, s := range config.Registry {
		if !catMap[s.Category] {
			catMap[s.Category] = true
			cats = append(cats, s.Category)
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Type here..."
	ti.Focus()

	return Model{
		categories:    cats,
		activeCatIdx:  0,
		activeItemIdx: 0,
		focusedPane:   0,
		statusMsg:     "",
		capturingMode: "none",
		input:         ti,
	}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle Key Capture Mode
		if m.capturingMode == "key" {
			if msg.String() == "esc" {
				m.statusMsg = "Canceled keybind change."
				m.capturingMode = "none"
				return m, nil
			}
			// Save the keybind
			s := m.getCurrentSetting()
			if s != nil {
				s.Value = msg.String()
				config.UpdateSettingValue(s.ID, msg.String())
				config.SaveSettings()
				m.statusMsg = "Saved keybind for " + s.Name + " -> " + msg.String()
			}
			m.capturingMode = "none"
			return m, nil
		}

		// Handle Text Input Mode
		if m.capturingMode == "text" {
			switch msg.String() {
			case "esc":
				m.statusMsg = "Canceled input."
				m.capturingMode = "none"
				return m, nil
			case "enter":
				s := m.getCurrentSetting()
				if s != nil {
					s.Value = m.input.Value()
					config.UpdateSettingValue(s.ID, m.input.Value())
					config.SaveSettings()
					m.statusMsg = "Saved " + s.Name
				}
				m.capturingMode = "none"
				return m, nil
			}
			
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Normal Navigation
		if msg.String() == "tab" || msg.String() == "l" || msg.String() == "right" {
			if m.focusedPane == 0 {
				m.focusedPane = 1
				m.activeItemIdx = 0
				return m, nil
			}
		}
		if msg.String() == "shift+tab" || msg.String() == "h" || msg.String() == "left" || msg.String() == "esc" || msg.String() == "backtab" {
			if m.focusedPane == 1 {
				m.focusedPane = 0
				return m, nil
			}
		}

		if m.focusedPane == 0 {
			switch msg.String() {
			case "up", "k":
				if m.activeCatIdx > 0 { m.activeCatIdx-- }
			case "down", "j":
				if m.activeCatIdx < len(m.categories)-1 { m.activeCatIdx++ }
			}
		} else if m.focusedPane == 1 {
			var catSettings []*config.Setting
			for i := range config.Registry {
				if config.Registry[i].Category == m.categories[m.activeCatIdx] {
					catSettings = append(catSettings, &config.Registry[i])
				}
			}

			switch msg.String() {
			case "up", "k":
				if m.activeItemIdx > 0 { m.activeItemIdx-- }
			case "down", "j":
				if m.activeItemIdx < len(catSettings)-1 { m.activeItemIdx++ }
			case "enter", "space":
				if len(catSettings) > 0 {
					s := catSettings[m.activeItemIdx]
					
					if s.Category == "Keybinds" {
						m.capturingMode = "key"
						m.statusMsg = "Press any key combination (or esc to cancel)..."
						return m, nil
					}

					if s.Type == config.TypeBool {
						if val, ok := s.Value.(bool); ok {
							s.Value = !val
						}
						config.UpdateSettingValue(s.ID, s.Value)
						config.SaveSettings()
						m.statusMsg = "Saved " + s.Name
					} else if s.Type == config.TypeSelect {
						if val, ok := s.Value.(string); ok {
							for i, opt := range s.Options {
								if opt == val {
									next := (i + 1) % len(s.Options)
									s.Value = s.Options[next]
									break
								}
							}
						}
						config.UpdateSettingValue(s.ID, s.Value)
						config.SaveSettings()
						if s.ID == "appearance.theme" {
							theme.SetTheme(s.Value.(string))
						}
						m.statusMsg = "Saved " + s.Name
					} else if s.Type == config.TypeString || s.Type == config.TypeInt {
						m.capturingMode = "text"
						m.input.SetValue(fmt.Sprintf("%v", s.Value))
						m.input.Focus()
						m.statusMsg = "Type new value and press Enter (or Esc to cancel)."
					}
				}
			}
		}
	}
	
	// Ensure input blinking continues if we are in text mode, though KeyMsg mostly handles it
	if m.capturingMode == "text" {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	
	return m, tea.Batch(cmds...)
}

func (m Model) getCurrentSetting() *config.Setting {
	var catSettings []*config.Setting
	for i := range config.Registry {
		if config.Registry[i].Category == m.categories[m.activeCatIdx] {
			catSettings = append(catSettings, &config.Registry[i])
		}
	}
	if len(catSettings) > 0 && m.activeItemIdx < len(catSettings) {
		return catSettings[m.activeItemIdx]
	}
	return nil
}

func (m Model) View() string {
	var sidebarItems string
	for i, c := range m.categories {
		style := lipgloss.NewStyle().Foreground(theme.Current.Dim).PaddingLeft(2)
		if i == m.activeCatIdx {
			style = lipgloss.NewStyle().Foreground(theme.Current.Accent).Bold(true).PaddingLeft(1)
			sidebarItems += style.Render("▶ " + c) + "\n"
		} else {
			sidebarItems += style.Render(c) + "\n"
		}
	}

	sidebarBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Width(25).
		Height(20).
		Padding(1).
		Render(lipgloss.NewStyle().Bold(true).Foreground(theme.Current.Text).Render("CATEGORIES") + "\n\n" + sidebarItems)

	if m.focusedPane != 0 {
		sidebarBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Current.Dim).
			Width(25).
			Height(20).
			Padding(1).
			Render(lipgloss.NewStyle().Bold(true).Foreground(theme.Current.Dim).Render("CATEGORIES") + "\n\n" + sidebarItems)
	}

	var contentItems string
	var catSettings []config.Setting
	for _, s := range config.Registry {
		if s.Category == m.categories[m.activeCatIdx] {
			catSettings = append(catSettings, s)
		}
	}

	for i, s := range catSettings {
		style := lipgloss.NewStyle().Foreground(theme.Current.Text)
		descStyle := lipgloss.NewStyle().Foreground(theme.Current.Dim)
		valStyle := lipgloss.NewStyle().Foreground(theme.Current.Accent).Bold(true)
		cursor := "  "
		
		if i == m.activeItemIdx && m.focusedPane == 1 {
			style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			cursor = "▶ "
		}

		var valStr string
		if m.capturingMode == "text" && i == m.activeItemIdx {
			valStr = m.input.View()
		} else if m.capturingMode == "key" && i == m.activeItemIdx {
			valStr = "[Press any key...]"
		} else if s.Type == config.TypeBool {
			if v, _ := s.Value.(bool); v {
				valStr = "[✓] Enabled "
			} else {
				valStr = "[ ] Disabled"
			}
		} else {
			valStr = fmt.Sprintf("[%v]", s.Value)
		}

		contentItems += fmt.Sprintf("%s%s\n  %s\n  Value: %s\n\n", cursor, style.Render(s.Name), descStyle.Render(s.Description), valStyle.Render(valStr))
	}

	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Dim).
		Width(60).
		Height(20).
		Padding(1).
		Render(contentItems)
	
	if m.focusedPane == 1 {
		contentBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Current.Primary).
			Width(60).
			Height(20).
			Padding(1).
			Render(contentItems)
	}

	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, contentBox)

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [left/right] Switch Panes  [up/down] Navigate  [ENTER] Toggle/Edit Value")
	statusBlock := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("\n" + m.statusMsg)

	finalContent := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("SYSTEM CONFIGURATION"),
		layout,
		statusBlock,
		controls,
	)

	return components.Card(finalContent, 95)
}

func (m Model) IsInputActive() bool {
	return m.capturingMode != "none"
}

func (m Model) Title() string { return "Settings" }
func (m Model) Icon() string { return "⚙️" }
