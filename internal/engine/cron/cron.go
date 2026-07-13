package cron

import (
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

type CronJob struct {
	User     string
	Schedule string
	Command  string
	Raw      string
}

// ListJobs fetches cron jobs for the current user and common system crontabs.
func (e *Engine) ListJobs() ([]CronJob, error) {
	var jobs []CronJob

	// Fetch user crontab
	out, _ := e.client.Run("crontab -l")
	jobs = append(jobs, parseCrontabOutput("user", out)...)

	// Fetch system crontabs (requires sudo or root usually, we'll just try to read them)
	sysOut, _ := e.client.Run("cat /etc/crontab /etc/cron.d/* 2>/dev/null")
	if sysOut != "" {
		jobs = append(jobs, parseSystemCrontabOutput(sysOut)...)
	}

	return jobs, nil
}

func parseCrontabOutput(user, output string) []CronJob {
	var jobs []CronJob
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Very basic cron parsing: 5 fields of time, rest is command
		parts := strings.Fields(line)
		if len(parts) >= 6 {
			schedule := strings.Join(parts[0:5], " ")
			cmd := strings.Join(parts[5:], " ")
			jobs = append(jobs, CronJob{
				User:     user,
				Schedule: schedule,
				Command:  cmd,
				Raw:      line,
			})
		}
	}
	return jobs
}

func parseSystemCrontabOutput(output string) []CronJob {
	var jobs []CronJob
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// System crontabs have an extra user field: 5 fields time, 1 field user, rest is command
		parts := strings.Fields(line)
		if len(parts) >= 7 {
			// Skip env vars like PATH=/sbin
			if strings.Contains(parts[0], "=") {
				continue
			}
			schedule := strings.Join(parts[0:5], " ")
			user := parts[5]
			cmd := strings.Join(parts[6:], " ")
			jobs = append(jobs, CronJob{
				User:     user,
				Schedule: schedule,
				Command:  cmd,
				Raw:      line,
			})
		}
	}
	return jobs
}

func (e *Engine) AddJob(schedule, cmd string) error {
	// Echo the new job into crontab
	// This is a naive implementation; ideally, we'd use a temporary file.
	// For simplicity, we just append to the current crontab: (crontab -l 2>/dev/null; echo "schedule cmd") | crontab -
	fullCmd := fmt.Sprintf("(crontab -l 2>/dev/null; echo \"%s %s\") | crontab -", schedule, cmd)
	_, err := e.client.Run(fullCmd)
	return err
}

func (e *Engine) DeleteJobRaw(raw string) error {
	// Safely delete a specific raw cron string from the user's crontab using grep -v
	escapedRaw := strings.ReplaceAll(raw, "\"", "\\\"")
	fullCmd := fmt.Sprintf("crontab -l | grep -v -F \"%s\" | crontab -", escapedRaw)
	_, err := e.client.Run(fullCmd)
	return err
}
