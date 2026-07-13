package snapshots

import (
	"fmt"
	sshlib "main/internal/ssh"
	"path/filepath"
	"strings"
	"time"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

type Snapshot struct {
	ID        string
	Target    string
	Timestamp string
	Size      string
}

// CreateSnapshot creates a backup of a specific file before it gets changed.
// It also auto-prunes so only the 5 most recent snapshots of that file are kept.
func (e *Engine) CreateSnapshot(targetPath string) (*Snapshot, error) {
	e.client.Run("mkdir -p /tmp/vortex_snapshots")

	timestamp := time.Now().Format("20060102_150405")

	// Encode path to avoid directory structure issues in /tmp/vortex_snapshots
	safePath := strings.ReplaceAll(targetPath, "/", "!")
	snapName := fmt.Sprintf("/tmp/vortex_snapshots/%s@%s.snap", safePath, timestamp)

	// Create snapshot
	cmd := fmt.Sprintf("cp -p %s %s 2>/dev/null", targetPath, snapName)
	_, err := e.client.Run(cmd)
	if err != nil {
		return nil, err
	}

	// Prune older ones automatically (keep the 5 most recent for this specific file)
	// We use ls -t to sort by time, tail to skip the first 5, and rm to delete the rest
	pruneCmd := fmt.Sprintf("ls -t /tmp/vortex_snapshots/%s@*.snap 2>/dev/null | tail -n +6 | xargs -I {} rm -f {}", safePath)
	e.client.Run(pruneCmd)

	sizeStr, _ := e.client.Run(fmt.Sprintf("du -sh %s | cut -f1", snapName))

	return &Snapshot{
		ID:        snapName,
		Target:    targetPath,
		Timestamp: timestamp,
		Size:      strings.TrimSpace(sizeStr),
	}, nil
}

// ListSnapshots scans the /tmp/vortex_snapshots directory for file snapshots
func (e *Engine) ListSnapshots() ([]Snapshot, error) {
	out, err := e.client.Run("ls -lh /tmp/vortex_snapshots/*.snap 2>/dev/null | awk '{print $5, $9}'")
	if err != nil {
		return nil, err
	}

	var snaps []Snapshot
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) == 2 {
			id := parts[1] // e.g. /tmp/vortex_snapshots/!etc!nginx!nginx.conf@20230101_150405.snap

			// Decode the target path from the ID
			base := filepath.Base(id)
			base = strings.TrimSuffix(base, ".snap")

			targetPath := "Unknown"
			timestamp := "Unknown"

			// Format is {safePath}@{timestamp}
			idx := strings.LastIndex(base, "@")
			if idx != -1 {
				safePath := base[:idx]
				timestamp = base[idx+1:]
				targetPath = strings.ReplaceAll(safePath, "!", "/")
			}

			snaps = append(snaps, Snapshot{
				ID:        id,
				Target:    targetPath,
				Timestamp: timestamp,
				Size:      parts[0],
			})
		}
	}
	return snaps, nil
}

// DeleteSnapshot removes a specific snapshot
func (e *Engine) DeleteSnapshot(id string) error {
	_, err := e.client.Run(fmt.Sprintf("rm -f %s", id))
	return err
}

// Rollback restores a snapshot over the original file
func (e *Engine) Rollback(id string, targetPath string) error {
	// Copy the snapshot back to the original target path
	_, err := e.client.Run(fmt.Sprintf("cp -p %s %s", id, targetPath))
	return err
}
