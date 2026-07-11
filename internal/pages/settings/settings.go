package settings

import (
	"fmt"

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
	isEditing     bool
	focusedPane   int // 0 = Sidebar, 1 = Items
	statusMsg     string
}

func New() Model {
	// Extract unique categories
	catMap := make(map[string]bool)
	var cats []string
	for _, s := range config.Registry {
		if !catMap[s.Category] {
			catMap[s.Category] = true
			cats = append(cats, s.Category)
		}
	}

	return Model{
		categories:    cats,
		activeCatIdx:  0,
		activeItemIdx: 0,
		focusedPane:   0,
		statusMsg:     "",
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Navigate panes
		if msg.String() == "tab" || msg.String() == "l" || msg.String() == "right" || msg.String() == "enter" {
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
			// Sidebar Navigation
			switch msg.String() {
			case "up", "k":
				if m.activeCatIdx > 0 {
					m.activeCatIdx--
				}
			case "down", "j":
				if m.activeCatIdx < len(m.categories)-1 {
					m.activeCatIdx++
				}
			}
		} else if m.focusedPane == 1 {
			// Item Navigation
			var catSettings []*config.Setting
			for i := range config.Registry {
				if config.Registry[i].Category == m.categories[m.activeCatIdx] {
					catSettings = append(catSettings, &config.Registry[i])
				}
			}

			switch msg.String() {
			case "up", "k":
				if m.activeItemIdx > 0 {
					m.activeItemIdx--
				}
			case "down", "j":
				if m.activeItemIdx < len(catSettings)-1 {
					m.activeItemIdx++
				}
			case "enter", "space":
				if len(catSettings) > 0 {
					s := catSettings[m.activeItemIdx]
					// Toggle logic
					if s.Type == config.TypeBool {
						if val, ok := s.Value.(bool); ok {
							s.Value = !val
						}
						config.SaveSettings()
						m.statusMsg = "Saved " + s.Name
					} else if s.Type == config.TypeSelect {
						// cycle options
						if val, ok := s.Value.(string); ok {
							for i, opt := range s.Options {
								if opt == val {
									next := (i + 1) % len(s.Options)
									s.Value = s.Options[next]
									break
								}
							}
						}
						config.SaveSettings()
						m.statusMsg = "Saved " + s.Name
					}
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	// Sidebar
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

	// Content Panel
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
		if s.Type == config.TypeBool {
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

	controls := lipgloss.NewStyle().Foreground(theme.Current.Dim).Render("\nControls: [left/right] Switch Panes  [up/down] Navigate  [ENTER] Toggle Value")
	statusBlock := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("\n" + m.statusMsg)

	finalContent := lipgloss.JoinVertical(lipgloss.Left,
		components.Title("SYSTEM CONFIGURATION"),
		layout,
		statusBlock,
		controls,
	)

	return components.Card(finalContent, 95)
}

func (m Model) Title() string { return "Settings" }
func (m Model) Icon() string { return "⚙️" }
