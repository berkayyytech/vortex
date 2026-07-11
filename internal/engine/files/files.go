package files

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

// ListDirectory fetches the contents of a directory on the remote server
func (e *Engine) ListDirectory(path string) ([]string, error) {
	out, err := e.client.Run(fmt.Sprintf("ls -1pA %s", path))
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var result []string
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result, nil
}
