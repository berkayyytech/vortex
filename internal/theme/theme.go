package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name    string
	Primary lipgloss.Color
	Accent  lipgloss.Color
	Dim         lipgloss.Color
	Bg          lipgloss.Color
	Text        lipgloss.Color
	HighlightBg lipgloss.Color
}

var (
	Catppuccin = Theme{Name: "Catppuccin", Primary: "86", Accent: "205", Dim: "240", Bg: "236", Text: "252", HighlightBg: "237"}
	Nord       = Theme{Name: "Nord", Primary: "81", Accent: "14", Dim: "238", Bg: "235", Text: "252", HighlightBg: "236"}
	TokyoNight = Theme{Name: "Tokyo Night", Primary: "111", Accent: "204", Dim: "239", Bg: "234", Text: "252", HighlightBg: "235"}
	Dracula    = Theme{Name: "Dracula", Primary: "141", Accent: "212", Dim: "238", Bg: "236", Text: "252", HighlightBg: "237"}

	Themes = []Theme{Catppuccin, Nord, TokyoNight, Dracula}
	Current = Catppuccin
)
