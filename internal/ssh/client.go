package ssh

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectedMsg is dispatched to Bubble Tea when a connection succeeds.
type ConnectedMsg struct {
	Client *Client
	Host   string
	User   string
}

type Client struct {
	conn *ssh.Client
}

// Connect establishes a secure connection to the remote Linux VPS.
func Connect(host, port, user, password, keyPath string) (*Client, error) {
	var authMethods []ssh.AuthMethod

	// Prefer SSH Key if provided
	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read private key: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("unable to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else if password != "" {
		// Fallback to password auth
		authMethods = append(authMethods, ssh.Password(password))
	} else {
		return nil, fmt.Errorf("no authentication method provided")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Safe for MVP testing
		Timeout:         5 * time.Second,
	}

	address := net.JoinHostPort(host, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	return &Client{conn: client}, nil
}

// Run executes a raw command on the remote server and returns its stdout.
func (c *Client) Run(cmd string) (string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf

	err = session.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %v", err)
	}

	return stdoutBuf.String(), nil
}

// DeployAndRunAgent compiles the vortex-agent for Linux, pushes it over SSH, and executes it.
func (c *Client) DeployAndRunAgent() ([]byte, error) {
	// 1. Cross-compile the agent locally
	agentPath := filepath.Join(os.TempDir(), "vortex-agent-linux")
	cmd := exec.Command("go", "build", "-o", agentPath, "./cmd/vortex-agent")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile agent: %v", err)
	}
	defer os.Remove(agentPath)

	// 2. Read the compiled binary
	binaryData, err := os.ReadFile(agentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled agent: %v", err)
	}

	// 3. Push binary over SSH
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create push session: %v", err)
	}
	
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	// Start the command that will receive the binary
	if err := session.Start("cat > /tmp/vortex-agent && chmod +x /tmp/vortex-agent"); err != nil {
		return nil, fmt.Errorf("failed to start remote receiver: %v", err)
	}

	// Stream the binary data
	if _, err := io.Copy(stdinPipe, bytes.NewReader(binaryData)); err != nil {
		return nil, fmt.Errorf("failed to stream binary: %v", err)
	}
	stdinPipe.Close()
	session.Wait() // Wait for transfer and chmod to finish

	// 4. Execute the agent and capture JSON payload
	return []byte(c.RunCommand("/tmp/vortex-agent payload")), nil
}

// RunCommand executes a raw command and returns stdout (simplified alias for Run)
func (c *Client) RunCommand(cmd string) string {
	out, _ := c.Run(cmd)
	return out
}

// Close terminates the SSH connection safely.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
