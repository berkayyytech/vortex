package pages

import (
	tea "github.com/charmbracelet/bubbletea"
	sshlib "main/internal/ssh"
)

// Page is the core interface that every module in Vortex must implement.
// By embedding tea.Model, every page manages its own state and event loop.
type Page interface {
	tea.Model
	Title() string
	Icon() string
}

// RunRemoteCmdMsg can be dispatched by any page to instruct the main router 
// to execute a shell command over the active SSH tunnel (e.g. restart docker).
type RunRemoteCmdMsg struct {
	Command string
}

// LogActivityMsg is broadcast to append an event to the Mission Control activity feed.
type LogActivityMsg struct {
	Message string
}

// RunRemoteQueryMsg allows pages to execute a command and handle the response async
type RunRemoteQueryMsg struct {
	Command         string
	ResponseHandler func(string) tea.Msg
}

// EngineReadyMsg is broadcasted to all pages when a server connects,
// allowing them to initialize their respective service engines (Docker, Systemd, etc.)
type EngineReadyMsg struct {
	Client *sshlib.Client
}

// Registry holds all registered pages for the application.
var registry []Page

// Register allows dynamic pages (like modules/plugins) to add themselves to the UI.
func Register(p Page) {
	registry = append(registry, p)
}

// GetAll returns all registered pages.
func GetAll() []Page {
	return registry
}
