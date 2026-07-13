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
	Name    string  `json:"name"`
	Runtime string  `json:"runtime"`
	PID     string  `json:"pid"`
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	User    string  `json:"user"`
}

// DetectApps scans the server for common runtimes running on listening ports
func (e *Engine) DetectApps() ([]App, error) {
	// A basic detection script that parses active ports and maps them to runtimes
	// Use top -b -n 2 to get real CPU usage deltas over 0.5s, rather than lifetime averages
	script := `
	LC_ALL=C COLUMNS=512 top -c -b -d 0.5 -n 2 | awk '
	/^top -/ {iter++}
	iter==2 && $1 ~ /^[0-9]+$/ {
		pid = $1
		user = $2
		cpu = $9
		mem = $10
		cmd = $12
		for(i=13; i<=NF; i++) cmd = cmd " " $i
		
		runtime = "System"
		if (cmd ~ /node/ || cmd ~ /server\.js/) runtime = "Node.js"
		else if (cmd ~ /python/) runtime = "Python"
		else if (cmd ~ /java/) runtime = "Java"
		else if (cmd ~ /go /) runtime = "Go"
		else if (cmd ~ /docker/) runtime = "Docker"
		else if (cmd ~ /pm2/ || cmd ~ /PM2/) runtime = "PM2"
		else if (cmd ~ /sshd/) runtime = "SSH"
		else if (cmd ~ /bash/ || cmd ~ /sh/) runtime = "Shell"
		
		# JSON escape
		gsub(/\\/, "\\\\", cmd)
		gsub(/"/, "\\\"", cmd)
		
		printf "{\"name\":\"%s\",\"runtime\":\"%s\",\"pid\":\"%s\",\"cpu\":%s,\"mem\":%s,\"user\":\"%s\"}\n", cmd, runtime, pid, cpu, mem, user
	}
	' | head -n 100
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
