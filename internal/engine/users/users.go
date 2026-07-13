package users

import (
	"encoding/base64"
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

type User struct {
	Username string
	HomeDir  string
	Shell    string
	UID      string
	Keys     []SSHKey
	Risky    bool
}

type SSHKey struct {
	Index   int
	Type    string
	Comment string
	Raw     string
}

func (e *Engine) GetUsers() ([]User, error) {
	out, err := e.client.Run("getent passwd")
	if err != nil {
		return nil, err
	}

	var users []User
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 7 {
			username := parts[0]
			uid := parts[2]
			homeDir := parts[5]
			shell := parts[6]

			// Filter out nologin and false shells, but keep root and valid shells
			if !strings.HasSuffix(shell, "nologin") && !strings.HasSuffix(shell, "false") && !strings.HasSuffix(shell, "sync") {
				keys, _ := e.GetUserKeys(username, homeDir)
				risky := false
				if username == "root" && len(keys) == 0 {
					// Root shouldn't strictly have keys if it's disabled, but this is a simple rule
					risky = true
				} else if len(keys) == 0 {
					risky = true
				}

				users = append(users, User{
					Username: username,
					HomeDir:  homeDir,
					Shell:    shell,
					UID:      uid,
					Keys:     keys,
					Risky:    risky,
				})
			}
		}
	}
	return users, nil
}

func (e *Engine) GetUserKeys(username, homeDir string) ([]SSHKey, error) {
	cmd := fmt.Sprintf("sudo cat %s/.ssh/authorized_keys", homeDir)
	out, err := e.client.Run(cmd)
	if err != nil {
		// likely doesn't exist
		return nil, err
	}

	var keys []SSHKey
	lines := strings.Split(strings.TrimSpace(out), "\n")
	idx := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			keyType := parts[0]
			comment := ""
			if len(parts) > 2 {
				comment = strings.Join(parts[2:], " ")
			}
			keys = append(keys, SSHKey{
				Index:   idx,
				Type:    keyType,
				Comment: comment,
				Raw:     line,
			})
			idx++
		}
	}
	return keys, nil
}

func (e *Engine) AddKey(username, homeDir, key string) error {
	cmd := fmt.Sprintf("sudo mkdir -p %[1]s/.ssh && sudo touch %[1]s/.ssh/authorized_keys && sudo chown -R %[2]s:%[2]s %[1]s/.ssh && sudo chmod 700 %[1]s/.ssh && sudo chmod 600 %[1]s/.ssh/authorized_keys", homeDir, username)
	if _, err := e.client.Run(cmd); err != nil {
		return fmt.Errorf("failed to prepare .ssh dir: %w", err)
	}

	// Use single quotes safely by avoiding single quotes in key (SSH keys shouldn't have them)
	cmd = fmt.Sprintf("echo '%s' | sudo tee -a %s/.ssh/authorized_keys > /dev/null", key, homeDir)
	_, err := e.client.Run(cmd)
	return err
}

func (e *Engine) RevokeKey(username, homeDir string, keyIndex int, keys []SSHKey) error {
	if keyIndex < 0 || keyIndex >= len(keys) {
		return fmt.Errorf("invalid key index")
	}

	var keepKeys []string
	for i, k := range keys {
		if i != keyIndex {
			keepKeys = append(keepKeys, k.Raw)
		}
	}

	newContent := strings.Join(keepKeys, "\n")
	if len(keepKeys) > 0 {
		newContent += "\n"
	}

	// base64 encode to safely write file content
	cmd := fmt.Sprintf("echo \"%s\" | base64 -d | sudo tee %s/.ssh/authorized_keys > /dev/null", base64Encode(newContent), homeDir)
	_, err := e.client.Run(cmd)
	return err
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
