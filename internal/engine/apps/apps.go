package apps

import (
	"encoding/json"
	sshlib "main/internal/ssh"
	"strings"
)

type Engine struct {
	client *sshlib.Client
}

func NewEngine(c *sshlib.Client) *Engine {
	return &Engine{client: c}
}

type App struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
	PID     string `json:"pid"`
	Port    string `json:"port"`
	Status  string `json:"status"`
}

// DetectApps scans the server for common runtimes running on listening ports
func (e *Engine) DetectApps() ([]App, error) {
	// A basic detection script that parses active ports and maps them to runtimes
	script := `
	ps -aux | awk 'NR>1 {
		user=$1; pid=$2; cpu=$3; mem=$4;
		command=$11;
		for(i=12; i<=NF; i++) command=command " " $i;
		print user, pid, cpu, mem, command
	}' | head -n 100 | while read user pid cpu mem command; do
		runtime="System"
		case "$command" in
			*node*|*server.js*) runtime="Node.js" ;;
			*python*) runtime="Python" ;;
			*java*) runtime="Java" ;;
			*go*) runtime="Go" ;;
			*docker*) runtime="Docker" ;;
			*pm2*|*PM2*) runtime="PM2" ;;
			*sshd*) runtime="SSH" ;;
			*bash*|*sh*) runtime="Shell" ;;
		esac
		
		# Escape quotes
		command=$(echo "$command" | sed 's/"/\\"/g' | cut -c 1-50)
		
		# Send as JSON
		echo "{\"name\":\"$command\",\"runtime\":\"$runtime\",\"pid\":\"$pid\",\"port\":\"$cpu% CPU\",\"status\":\"$user\"}"
	done
	`
	out, err := e.client.Run(script)
	if err != nil {
		return nil, err
	}

	var apps []App
	lines := strings.Split(strings.TrimSpace(out), "\n")
	
	// Use a map to deduplicate by PID
	appMap := make(map[string]App)
	
	for _, l := range lines {
		if l == "" {
			continue
		}
		var app App
		if err := json.Unmarshal([]byte(l), &app); err == nil {
			if _, exists := appMap[app.PID]; !exists {
				appMap[app.PID] = app
			}
		}
	}

	for _, a := range appMap {
		apps = append(apps, a)
	}

	return apps, nil
}

// StopProcess gracefully kills a process
func (e *Engine) StopProcess(pid string) error {
	script := `
	if command -v pm2 >/dev/null 2>&1 && command -v node >/dev/null 2>&1; then
		PM2_ID=$(node -e "try { const p = require('child_process').execSync('pm2 jlist 2>/dev/null').toString(); JSON.parse(p).forEach(a => { if(a.pid == ` + pid + `) console.log(a.pm_id) }) } catch(e) {}")
		if [ ! -z "$PM2_ID" ]; then
			pm2 stop "$PM2_ID"
			exit 0
		fi
	fi
	kill -15 ` + pid + `
	`
	_, err := e.client.Run(script)
	return err
}

// KillProcess forcefully kills a process
func (e *Engine) KillProcess(pid string) error {
	script := `
	if command -v pm2 >/dev/null 2>&1 && command -v node >/dev/null 2>&1; then
		PM2_ID=$(node -e "try { const p = require('child_process').execSync('pm2 jlist 2>/dev/null').toString(); JSON.parse(p).forEach(a => { if(a.pid == ` + pid + `) console.log(a.pm_id) }) } catch(e) {}")
		if [ ! -z "$PM2_ID" ]; then
			pm2 delete "$PM2_ID"
			exit 0
		fi
	fi
	kill -9 ` + pid + `
	`
	_, err := e.client.Run(script)
	return err
}
