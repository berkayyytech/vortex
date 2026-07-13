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

type FileInfo struct {
	Name        string
	IsDir       bool
	Size        string
	Permissions string
	Owner       string
	Modified    string
}

// ListDirectory fetches the contents of a directory on the remote server
func (e *Engine) ListDirectory(path string) ([]FileInfo, error) {
	out, err := e.client.Run(fmt.Sprintf("ls -lhpA %q", path))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var result []FileInfo
	for _, l := range lines {
		if l == "" || strings.HasPrefix(l, "total ") {
			continue
		}
		
		parts := strings.Fields(l)
		if len(parts) < 9 {
			continue
		}
		
		permissions := parts[0]
		owner := parts[2]
		
		size := parts[4]
		modified := parts[5] + " " + parts[6] + " " + parts[7]
		name := strings.Join(parts[8:], " ")
		
		isDir := strings.HasPrefix(permissions, "d") || strings.HasSuffix(name, "/")
		if isDir && !strings.HasSuffix(name, "/") {
			name += "/"
		}
		
		result = append(result, FileInfo{
			Name:        name,
			IsDir:       isDir,
			Size:        size,
			Permissions: permissions,
			Owner:       owner,
			Modified:    modified,
		})
	}
	return result, nil
}

func (e *Engine) ReadFile(path string) (string, error) {
	return e.client.Run(fmt.Sprintf("sudo cat %q", path))
}

func (e *Engine) WriteFile(path, content string) error {
	// escape single quotes for shell
	escaped := strings.ReplaceAll(content, "'", "'\"'\"'")
	_, err := e.client.Run(fmt.Sprintf("echo '%s' | sudo tee %q > /dev/null", escaped, path))
	return err
}

func (e *Engine) Rename(oldPath, newPath string) error {
	_, err := e.client.Run(fmt.Sprintf("mv %q %q", oldPath, newPath))
	return err
}

func (e *Engine) Delete(path string) error {
	_, err := e.client.Run(fmt.Sprintf("rm -rf %q", path))
	return err
}

func (e *Engine) Copy(src, dst string) error {
	_, err := e.client.Run(fmt.Sprintf("cp -r %q %q", src, dst))
	return err
}

func (e *Engine) Move(src, dst string) error {
	_, err := e.client.Run(fmt.Sprintf("mv %q %q", src, dst))
	return err
}

func (e *Engine) Chmod(path, mode string) error {
	_, err := e.client.Run(fmt.Sprintf("chmod %q %q", mode, path))
	return err
}
