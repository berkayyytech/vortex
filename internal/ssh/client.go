package ssh

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectedMsg is dispatched to Bubble Tea when a connection succeeds.
type ConnectedMsg struct {
	Client *Client
	Host   string
	Port   string
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

	// Start KeepAlive to prevent server from disconnecting inactive client
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return // connection closed or dead
			}
		}
	}()

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
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %v, stderr: %s", err, stderrBuf.String())
	}

	return stdoutBuf.String(), nil
}

// DeployAndRunAgent compiles the vortex-agent for Linux, pushes it over SSH, and executes it.
func (c *Client) DeployAndRunAgent() ([]byte, error) {
	// 0. Detect target OS
	targetOS := "linux"
	
	unameOut, _ := c.Run("uname -s")
	if strings.Contains(strings.ToLower(unameOut), "windows") || unameOut == "" {
		osOut, _ := c.Run("echo %OS%")
		if strings.Contains(strings.ToUpper(osOut), "WINDOWS") {
			targetOS = "windows"
		}
	}

	// 1. Cross-compile the agent locally
	// Find the project root by looking for cmd/vortex-agent
	agentSrcPath := "./cmd/vortex-agent"
	if _, err := os.Stat(agentSrcPath); os.IsNotExist(err) {
		// If running from cmd/vps-manager, go up one or two levels
		if _, err := os.Stat("../vortex-agent"); err == nil {
			agentSrcPath = "../vortex-agent"
		} else if _, err := os.Stat("../../cmd/vortex-agent"); err == nil {
			agentSrcPath = "../../cmd/vortex-agent"
		}
	}

	agentPath := filepath.Join(os.TempDir(), "vortex-agent-build")
	cmd := exec.Command("go", "build", "-o", agentPath, agentSrcPath)
	cmd.Env = append(os.Environ(), "GOOS="+targetOS, "GOARCH=amd64", "CGO_ENABLED=0")
	var compileErr bytes.Buffer
	cmd.Stderr = &compileErr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to compile agent: %v, stderr: %s", err, compileErr.String())
	}
	defer os.Remove(agentPath)

	// 2. Read the compiled binary
	binaryData, err := os.ReadFile(agentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compiled agent: %v", err)
	}

	// 3. Push binary over SSH via base64 chunking
	// We use base64 over stdin to avoid command line length limits on Windows,
	// and to ensure clean binary transfer without relying on cat.
	session, err := c.conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create push session: %v", err)
	}
	
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	var deployCmd string
	if targetOS == "windows" {
		deployCmd = `powershell -Command "$b = [System.Text.StringBuilder]::new(); while ($line = [Console]::ReadLine()) { $b.Append($line) > $null }; [IO.File]::WriteAllBytes('vortex-agent.exe', [Convert]::FromBase64String($b.ToString()))"`
	} else {
		deployCmd = `cat > ./vortex-agent && chmod +x ./vortex-agent`
	}

	if err := session.Start(deployCmd); err != nil {
		return nil, fmt.Errorf("failed to start remote receiver: %v", err)
	}

	if targetOS == "windows" {
		// Windows still uses base64 for now
		encoder := base64.NewEncoder(base64.StdEncoding, stdinPipe)
		io.Copy(encoder, bytes.NewReader(binaryData))
		encoder.Close()
	} else {
		// Linux uses raw binary over stdin
		io.Copy(stdinPipe, bytes.NewReader(binaryData))
	}
	stdinPipe.Close()
	
	session.Wait() // Wait for transfer and chmod to finish

	// 4. Execute the agent and capture JSON payload
	if targetOS == "windows" {
		return []byte(c.RunCommand(".\\vortex-agent.exe payload")), nil
	}
	return []byte(c.RunCommand("./vortex-agent payload")), nil
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
