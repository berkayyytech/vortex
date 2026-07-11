package systemd

import (
	"fmt"
	sshlib "main/internal/ssh"
	"strings"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

func (e *Engine) RestartService(name string) error {
	_, err := e.client.Run(fmt.Sprintf("sudo systemctl restart %s", name))
	return err
}

func (e *Engine) StopService(name string) error {
	_, err := e.client.Run(fmt.Sprintf("sudo systemctl stop %s", name))
	return err
}

func (e *Engine) ListServices() (string, error) {
	// Native systemd listing
	out, err := e.client.Run("systemctl list-units --type=service --no-pager --no-legend")
	if err != nil {
		// Fallback for Alpine/Test containers without systemd
		if strings.Contains(err.Error(), "systemctl: command not found") {
			return "", fmt.Errorf("systemd not available on this OS")
		}
		return "", err
	}
	return out, nil
}
