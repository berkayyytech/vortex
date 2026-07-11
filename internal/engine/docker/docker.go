package docker

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

// RestartContainer restarts a specific container by ID or name.
func (e *Engine) RestartContainer(id string) error {
	_, err := e.client.Run(fmt.Sprintf("docker restart %s", id))
	return err
}

// StopContainer stops a specific container.
func (e *Engine) StopContainer(id string) error {
	_, err := e.client.Run(fmt.Sprintf("docker stop %s", id))
	return err
}

// StartContainer starts a specific container.
func (e *Engine) StartContainer(id string) error {
	_, err := e.client.Run(fmt.Sprintf("docker start %s", id))
	return err
}

// ListContainers returns the JSON output of running containers.
func (e *Engine) ListContainers() (string, error) {
	// Real-world: docker ps --format "{{json .}}"
	out, err := e.client.Run(`docker ps --format '{"id":"{{.ID}}","name":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}","state":"{{.State}}"}'`)
	if err != nil {
		return "", err
	}
	
	// Format to valid JSON array
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return "[" + strings.Join(lines, ",") + "]", nil
}
