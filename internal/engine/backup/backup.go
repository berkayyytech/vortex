package backup

import (
	"fmt"
	sshlib "main/internal/ssh"
	"strings"
	"time"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

type BackupJob struct {
	ID        string
	Type      string // "File", "Docker Volume", "Database"
	Target    string
	Timestamp string
	Size      string
	Status    string // "Success", "Failed", "In Progress"
}

// CreateBackup creates a tar.gz archive of the target directory on the remote server
func (e *Engine) CreateBackup(targetPath string, destPath string, backupType string) (*BackupJob, error) {
	// Ensure destPath exists
	_, err := e.client.Run(fmt.Sprintf("mkdir -p %s", destPath))
	if err != nil {
		return &BackupJob{Status: "Failed"}, err
	}

	timestamp := time.Now().Format("20060102_150405")
	archiveName := fmt.Sprintf("%s/vortex_backup_%s.tar.gz", strings.TrimRight(destPath, "/"), timestamp)
	
	cmd := fmt.Sprintf("tar -czf %s %s 2>/dev/null", archiveName, targetPath)
	_, err = e.client.Run(cmd)
	if err != nil {
		return &BackupJob{Status: "Failed"}, err
	}

	sizeStr, _ := e.client.Run(fmt.Sprintf("du -sh %s | cut -f1", archiveName))

	return &BackupJob{
		ID:        archiveName,
		Type:      backupType,
		Target:    targetPath,
		Timestamp: timestamp,
		Size:      strings.TrimSpace(sizeStr),
		Status:    "Success",
	}, nil
}

// ListBackups scans the destination directory for vortex backups
func (e *Engine) ListBackups(destPath string) ([]BackupJob, error) {
	cmd := fmt.Sprintf("ls -lh %s/vortex_backup_*.tar.gz 2>/dev/null | awk '{print $5, $9}'", strings.TrimRight(destPath, "/"))
	out, err := e.client.Run(cmd)
	if err != nil {
		return nil, err
	}

	var jobs []BackupJob
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		if l == "" {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) == 2 {
			jobs = append(jobs, BackupJob{
				ID:        parts[1],
				Type:      "Archive",
				Target:    parts[1],
				Timestamp: "Stored",
				Size:      parts[0],
				Status:    "Success",
			})
		}
	}
	return jobs, nil
}

// DeleteBackup removes a backup archive from the server
func (e *Engine) DeleteBackup(id string) error {
	_, err := e.client.Run(fmt.Sprintf("rm -f %s", id))
	return err
}

// RestoreBackup restores a backup archive to the root directory
func (e *Engine) RestoreBackup(id string) error {
	_, err := e.client.Run(fmt.Sprintf("tar -xzf %s -C /", id))
	return err
}

// GetStorageUsage returns the total storage used by the backups in destPath
func (e *Engine) GetStorageUsage(destPath string) (string, error) {
	out, err := e.client.Run(fmt.Sprintf("du -sh %s 2>/dev/null | cut -f1", destPath))
	if err != nil || strings.TrimSpace(out) == "" {
		return "0B", err
	}
	return strings.TrimSpace(out), nil
}
