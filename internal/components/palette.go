package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"main/internal/theme"
)

type Command struct {
	Name        string
	Description string
	Action      func() tea.Cmd
}

type Palette struct {
	Active   bool
	query    string
	commands []Command
	filtered []Command
	cursor   int
}

func NewPalette() Palette {
	return Palette{
		Active:   false,
		query:    "",
		commands: []Command{},
		filtered: []Command{},
		cursor:   0,
	}
}

// RegisterCommand allows pages to register global commands
func (p *Palette) RegisterCommand(cmd Command) {
	p.commands = append(p.commands, cmd)
	p.filter()
}

func (p *Palette) filter() {
	if p.query == "" {
		p.filtered = p.commands
	} else {
		p.filtered = []Command{}
		for _, c := range p.commands {
			if strings.Contains(strings.ToLower(c.Name), strings.ToLower(p.query)) ||
				strings.Contains(strings.ToLower(c.Description), strings.ToLower(p.query)) {
				p.filtered = append(p.filtered, c)
			}
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

func (p *Palette) Update(msg tea.Msg) (Palette, tea.Cmd) {
	if !p.Active {
		return *p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+p":
			p.Active = false
			p.query = ""
			return *p, nil
		case "up":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			}
		case "enter":
			if len(p.filtered) > 0 {
				cmd := p.filtered[p.cursor]
				p.Active = false
				p.query = ""
				if cmd.Action != nil {
					return *p, cmd.Action()
				}
			}
		case "backspace":
			if len(p.query) > 0 {
				p.query = p.query[:len(p.query)-1]
				p.filter()
			}
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				p.query += msg.String()
				p.filter()
			}
		}
	}
	return *p, nil
}

func (p *Palette) View() string {
	if !p.Active {
		return ""
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		Render("🔍 COMMAND PALETTE")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Accent).
		Padding(0, 1).
		Width(60).
		Render("> " + p.query + "█")

	var list string
	for i, c := range p.filtered {
		if i >= 10 { // limit to 10 items
			break
		}
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(theme.Current.Text)
		descStyle := lipgloss.NewStyle().Foreground(theme.Current.Dim)
		
		if i == p.cursor {
			cursor = "▶ "
			style = lipgloss.NewStyle().Foreground(theme.Current.Primary).Bold(true)
			descStyle = lipgloss.NewStyle().Foreground(theme.Current.Accent)
		}
		list += fmt.Sprintf("%s %s %s\n", cursor, style.Render(c.Name), descStyle.Render("- "+c.Description))
	}

	if len(p.filtered) == 0 {
		list = "  No commands found."
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Background(lipgloss.Color("0")).
		Padding(1, 2).
		Margin(1, 0)

	content := lipgloss.JoinVertical(lipgloss.Left, header, inputBox, "\n", list)
	return box.Render(content)
}
