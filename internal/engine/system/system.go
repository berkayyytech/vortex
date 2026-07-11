package system

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"main/internal/agent"
	sshlib "main/internal/ssh"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

// FetchPayload is a tea.Cmd that asynchronously polls the remote agent
// for live system telemetry (CPU, RAM, Docker, Services) without blocking the UI.
func (e *Engine) FetchPayload(includeLogs bool) tea.Cmd {
	return func() tea.Msg {
		out, err := e.client.Run("/tmp/vortex-agent payload")
		if err != nil {
			// Return a specialized error message if the agent is down
			return agent.PayloadErrorMsg{Err: err}
		}

		var payload agent.Payload
		if err := json.Unmarshal([]byte(out), &payload); err != nil {
			return agent.PayloadErrorMsg{Err: err}
		}

		if includeLogs {
			// Fetch live logs if explicitly requested by the UI state
			logsOut, err := e.client.Run("journalctl -n 25 --no-pager")
			if err == nil {
				payload.Logs = logsOut
			}
		}

		return payload
	}
}

// Tick returns a tea.Cmd that waits for the specified duration before triggering the next poll.
func Tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return agent.TickMsg(t)
	})
}
