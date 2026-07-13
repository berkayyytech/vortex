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
	RAM     string  `json:"ram"`
	User    string  `json:"user"`
}

// DetectApps scans the server for common runtimes running on listening ports
func (e *Engine) DetectApps() ([]App, error) {
	// Use ps and /proc/stat to compute CPU delta precisely over an interval
	script := `
	cat /proc/[0-9]*/stat 2>/dev/null > /tmp/.v1
	read -r cpu user nice system idle iowait irq softirq steal guest guest_nice < /proc/stat
	total1=$((user + nice + system + idle + iowait + irq + softirq + steal))

	sleep 0.5

	cat /proc/[0-9]*/stat 2>/dev/null > /tmp/.v2
	read -r cpu user nice system idle iowait irq softirq steal guest guest_nice < /proc/stat
	total2=$((user + nice + system + idle + iowait + irq + softirq + steal))
	td=$((total2 - total1))

	cores=$(nproc 2>/dev/null || echo 1)

	LC_ALL=en_US.UTF-8 ps -eo pid,user,%mem,rss,args --no-headers | awk -v td="$td" -v cores="$cores" '
	BEGIN {
		while ((getline < "/tmp/.v1") > 0) p1[$1] = $14 + $15
		close("/tmp/.v1")
		while ((getline < "/tmp/.v2") > 0) p2[$1] = $14 + $15
		close("/tmp/.v2")
	}
	{
		pid = $1
		user = $2
		mem = $3
		rss = $4
		cmd = $5
		for (i=6; i<=NF; i++) cmd = cmd " " $i
		
		cpu = 0.0
		if (pid in p1 && pid in p2 && td > 0) {
			diff = p2[pid] - p1[pid]
			cpu = (diff / td) * 100.0 * cores
		}
		
		ram_mb = rss / 1024.0
		ram_str = sprintf("%.1f MB", ram_mb)

		runtime = "System"
		if (cmd ~ /node/ || cmd ~ /server\.js/) runtime = "Node.js"
		else if (cmd ~ /python/) runtime = "Python"
		else if (cmd ~ /java/) runtime = "Java"
		else if (cmd ~ /go /) runtime = "Go"
		else if (cmd ~ /docker/) runtime = "Docker"
		else if (cmd ~ /pm2/ || cmd ~ /PM2/) runtime = "PM2"
		else if (cmd ~ /sshd/) runtime = "SSH"
		else if (cmd ~ /bash/ || cmd ~ /sh/) runtime = "Shell"
		
		gsub(/\\/, "\\\\", cmd)
		gsub(/"/, "\\\"", cmd)
		
		printf "{\"name\":\"%s\",\"runtime\":\"%s\",\"pid\":\"%s\",\"cpu\":%s,\"mem\":%s,\"ram\":\"%s\",\"user\":\"%s\"}\n", cmd, runtime, pid, cpu, mem, ram_str, user
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
