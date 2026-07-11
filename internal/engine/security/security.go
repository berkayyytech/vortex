package security

import (
	sshlib "main/internal/ssh"
	"strings"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

type SecurityReport struct {
	UFWStatus         string
	RootLoginEnabled  bool
	PasswordAuthEnabled bool
}

// RunAudit performs a quick audit of the server's security posture
func (e *Engine) RunAudit() (*SecurityReport, error) {
	report := &SecurityReport{}

	// Check UFW
	out, err := e.client.Run("sudo ufw status")
	if err == nil {
		if strings.Contains(out, "Status: active") {
			report.UFWStatus = "Active"
		} else {
			report.UFWStatus = "Inactive"
		}
	} else {
		report.UFWStatus = "Not Installed / Error"
	}

	// Check SSH Config
	sshdOut, _ := e.client.Run("cat /etc/ssh/sshd_config")
	if strings.Contains(sshdOut, "PermitRootLogin yes") {
		report.RootLoginEnabled = true
	} else {
		report.RootLoginEnabled = false // safe default or explicitly no
	}

	if strings.Contains(sshdOut, "PasswordAuthentication yes") {
		report.PasswordAuthEnabled = true
	} else if strings.Contains(sshdOut, "PasswordAuthentication no") {
		report.PasswordAuthEnabled = false
	} else {
		report.PasswordAuthEnabled = true // default sshd config usually allows passwords
	}

	return report, nil
}
