package secrets

import (
	"encoding/base64"
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

// SearchEnvFiles looks for .env files on the server (max depth 4)
func (e *Engine) SearchEnvFiles() ([]string, error) {
	out, err := e.client.Run("find / -maxdepth 4 -name '*.env' 2>/dev/null | grep -v 'Permission denied' || true")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var result []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			result = append(result, l)
		}
	}
	return result, nil
}

func (e *Engine) ReadFile(path string) (string, error) {
	return e.client.Run(fmt.Sprintf("cat %q", path))
}

func (e *Engine) WriteFile(path, content string) error {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	_, err := e.client.Run(fmt.Sprintf("echo '%s' | base64 -d > %q", encoded, path))
	return err
}

// SyncToDatabaseManager returns nil, simulating integration with Database Manager
func (e *Engine) SyncToDatabaseManager(path string) error {
	return nil
}
