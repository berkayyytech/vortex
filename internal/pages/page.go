package pages

import tea "github.com/charmbracelet/bubbletea"

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

// RunRemoteQueryMsg allows pages to execute a command and handle the response async
type RunRemoteQueryMsg struct {
	Command         string
	ResponseHandler func(string) tea.Msg
}
