package logs

import (
	"fmt"
	"strings"
	sshlib "main/internal/ssh"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

// FetchSystemdLogs fetches the latest lines from journalctl
func (e *Engine) FetchSystemdLogs(lines int) (string, error) {
	out, err := e.client.Run(fmt.Sprintf("journalctl -n %d --no-pager", lines))
	if err != nil {
		if strings.Contains(err.Error(), "command not found") || strings.Contains(err.Error(), "127") {
			return "Systemd (journalctl) is not available on this server or container.", nil
		}
		return "", err
	}
	return out, nil
}

// FetchDockerLogs fetches the latest lines for a specific container
func (e *Engine) FetchDockerLogs(containerID string, lines int) (string, error) {
	return e.client.Run(fmt.Sprintf("docker logs --tail %d %s", lines, containerID))
}

// FetchFileLogs fetches the latest lines from a specific file (e.g. /var/log/syslog)
func (e *Engine) FetchFileLogs(filepath string, lines int) (string, error) {
	return e.client.Run(fmt.Sprintf("tail -n %d %s", lines, filepath))
}
