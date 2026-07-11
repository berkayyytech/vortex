package services

import (
	"fmt"
	"os/exec"
	"strings"
)

type Service struct {
	Name   string
	Status string
	Uptime string
}

func GetServices() []Service {
	// Execute systemctl to get real services
	out, err := exec.Command("systemctl", "list-units", "--type=service", "--no-pager", "--no-legend").Output()
	if err != nil {
		return []Service{
			{Name: "systemd not available on this OS", Status: "error", Uptime: "-"},
		}
	}

	lines := strings.Split(string(out), "\n")
	var services []Service
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// e.g. "nginx.service loaded active running Nginx Web Server"
			services = append(services, Service{
				Name:   fields[0],
				Status: fields[3], // we use substate as status (e.g. running, exited, failed)
				Uptime: "-",
			})
		}
	}
	return services
}

func FormatServices(services []Service) string {
	res := fmt.Sprintf("%-20s %-20s %-10s\n", "NAME", "STATUS", "UPTIME")
	res += "────────────────────────────────────────────────────\n"
	for _, s := range services {
		res += fmt.Sprintf("%-20s %-20s %-10s\n", s.Name, s.Status, s.Uptime)
	}
	return res
}
