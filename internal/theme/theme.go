package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name    string
	Primary     lipgloss.Color
	Accent      lipgloss.Color
	Dim         lipgloss.Color
	Bg          lipgloss.Color
	Text        lipgloss.Color
	HighlightBg lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	Info        lipgloss.Color
	Network     lipgloss.Color
}

var (
	Catppuccin = Theme{Name: "Catppuccin", Primary: "86", Accent: "205", Dim: "240", Bg: "236", Text: "252", HighlightBg: "237", Success: "40", Warning: "214", Error: "196", Info: "39", Network: "51"}
	Nord       = Theme{Name: "Nord", Primary: "81", Accent: "14", Dim: "238", Bg: "235", Text: "252", HighlightBg: "236", Success: "114", Warning: "220", Error: "203", Info: "111", Network: "117"}
	TokyoNight = Theme{Name: "Tokyo Night", Primary: "111", Accent: "204", Dim: "239", Bg: "234", Text: "252", HighlightBg: "235", Success: "114", Warning: "220", Error: "203", Info: "111", Network: "117"}
	Dracula    = Theme{Name: "Dracula", Primary: "141", Accent: "212", Dim: "238", Bg: "236", Text: "252", HighlightBg: "237", Success: "84", Warning: "228", Error: "196", Info: "117", Network: "87"}
	Gruvbox    = Theme{Name: "Gruvbox", Primary: "214", Accent: "142", Dim: "241", Bg: "235", Text: "223", HighlightBg: "237", Success: "142", Warning: "214", Error: "167", Info: "109", Network: "108"}
	Monokai    = Theme{Name: "Monokai", Primary: "197", Accent: "148", Dim: "240", Bg: "235", Text: "231", HighlightBg: "237", Success: "148", Warning: "208", Error: "197", Info: "81", Network: "81"}
	Matrix     = Theme{Name: "Matrix", Primary: "46", Accent: "10", Dim: "22", Bg: "16", Text: "46", HighlightBg: "22", Success: "46", Warning: "226", Error: "196", Info: "51", Network: "51"}
	Cyberpunk  = Theme{Name: "Cyberpunk", Primary: "201", Accent: "118", Dim: "240", Bg: "233", Text: "255", HighlightBg: "237", Success: "118", Warning: "226", Error: "196", Info: "51", Network: "51"}
	RosePine   = Theme{Name: "Rose Pine", Primary: "211", Accent: "189", Dim: "238", Bg: "234", Text: "252", HighlightBg: "236", Success: "114", Warning: "220", Error: "203", Info: "111", Network: "117"}
	GitHubDark = Theme{Name: "GitHub Dark", Primary: "39", Accent: "212", Dim: "241", Bg: "232", Text: "252", HighlightBg: "235", Success: "77", Warning: "214", Error: "196", Info: "39", Network: "51"}

	Themes = []Theme{Catppuccin, Nord, TokyoNight, Dracula, Gruvbox, Monokai, Matrix, Cyberpunk, RosePine, GitHubDark}
	Current = Catppuccin
)

func SetTheme(name string) {
	for _, t := range Themes {
		if t.Name == name {
			Current = t
			return
		}
	}
}

