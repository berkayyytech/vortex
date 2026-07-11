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
	ss -tlnp 2>/dev/null | grep -v 'State' | awk '{print $4, $6}' | while read port_info proc_info; do
		port="${port_info##*:}"
		pid=$(echo "$proc_info" | grep -oP 'pid=\K[0-9]+')
		proc_name=$(echo "$proc_info" | grep -oP 'users:\(\("\K[^"]+')
		
		if [ ! -z "$pid" ] && [ ! -z "$proc_name" ]; then
			runtime="Unknown"
			case "$proc_name" in
				node*) runtime="Node.js" ;;
				python*) runtime="Python" ;;
				java) runtime="Java" ;;
				go*) runtime="Go" ;;
				docker-proxy) runtime="Docker App" ;;
				pm2*) runtime="PM2" ;;
			esac
			
			echo "{\"name\":\"$proc_name\",\"runtime\":\"$runtime\",\"pid\":\"$pid\",\"port\":\"$port\",\"status\":\"running\"}"
		fi
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
	_, err := e.client.Run("kill -15 " + pid)
	return err
}

// KillProcess forcefully kills a process
func (e *Engine) KillProcess(pid string) error {
	_, err := e.client.Run("kill -9 " + pid)
	return err
}
