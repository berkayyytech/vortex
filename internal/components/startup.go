package components

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Startup struct {
	Active      bool
	Frame       int
	MaxFrames   int
	Steps       []string
	CurrentStep int
}

func NewStartup() Startup {
	return Startup{
		Active:    true,
		Frame:     0,
		MaxFrames: 45, // approx 2.25 seconds
		Steps: []string{
			"✓ Loading configuration",
			"✓ Connecting to servers",
			"✓ Loading themes",
			"✓ Detecting operating system",
			"✓ Initializing dashboard",
			"✓ Establishing SSH sessions",
		},
		CurrentStep: 0,
	}
}

type TickStartupMsg time.Time

func TickStartup() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return TickStartupMsg(t)
	})
}

func (s *Startup) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case TickStartupMsg:
		s.Frame++
		if s.Frame%6 == 0 && s.CurrentStep < len(s.Steps) {
			s.CurrentStep++
		}
		if s.Frame >= s.MaxFrames {
			s.Active = false
			return nil
		}
		return TickStartup()
	}
	return nil
}

func (s *Startup) View(width, height int) string {
	cyan := lipgloss.Color("86")
	blue := lipgloss.Color("33")
	green := lipgloss.Color("46")
	
	// Fade effect logic
	logoColor := cyan
	if s.Frame < 10 {
		logoColor = lipgloss.Color("23")
	} else if s.Frame < 20 {
		logoColor = blue
	}

	logoStyle := lipgloss.NewStyle().Foreground(logoColor).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(green).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	logo := `
██████╗ 
██╔══██╗
██████╔╝
██╔══██╗
██║  ██║`

	renderedLogo := logoStyle.Render(logo)
	renderedTitle := titleStyle.Render("V O R T E X")
	divider := dimStyle.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	initText := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("Initializing Mission Control...")

	var steps []string
	for i := 0; i < s.CurrentStep; i++ {
		steps = append(steps, lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(s.Steps[i]))
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		renderedLogo,
		"",
		renderedTitle,
		"",
		divider,
		"",
		initText,
		"",
		strings.Join(steps, "\n\n"),
		"",
		divider,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
