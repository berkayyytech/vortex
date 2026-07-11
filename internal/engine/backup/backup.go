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
func (e *Engine) CreateBackup(targetPath string, backupType string) (*BackupJob, error) {
	timestamp := time.Now().Format("20060102_150405")
	archiveName := fmt.Sprintf("/tmp/vortex_backup_%s.tar.gz", timestamp)
	
	// Run tar command asynchronously or wait (simplification: wait for small dirs)
	cmd := fmt.Sprintf("tar -czf %s %s 2>/dev/null", archiveName, targetPath)
	_, err := e.client.Run(cmd)
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

// ListBackups scans the /tmp directory for vortex backups
func (e *Engine) ListBackups() ([]BackupJob, error) {
	out, err := e.client.Run("ls -lh /tmp/vortex_backup_*.tar.gz 2>/dev/null | awk '{print $5, $9}'")
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
