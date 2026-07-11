package pages

import tea "github.com/charmbracelet/bubbletea"

// Page is the core interface that every module in Vortex must implement.
// By embedding tea.Model, every page manages its own state and event loop.
type Page interface {
	tea.Model
	Title() string
	Icon() string
}
