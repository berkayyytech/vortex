package services

import "fmt"

type Service struct {
	Name   string
	Status string
	Uptime string
}

func GetServices() []Service {
	// In production, execute systemctl or use dbus
	return []Service{
		{Name: "nginx.service", Status: "active (running)", Uptime: "12d 4h"},
		{Name: "postgresql.service", Status: "active (running)", Uptime: "45d 1h"},
		{Name: "docker.service", Status: "active (running)", Uptime: "5d 10h"},
		{Name: "redis.service", Status: "failed", Uptime: "-"},
		{Name: "ssh.service", Status: "active (running)", Uptime: "60d 2h"},
	}
}

func FormatServices(services []Service) string {
	res := fmt.Sprintf("%-20s %-20s %-10s\n", "NAME", "STATUS", "UPTIME")
	res += "────────────────────────────────────────────────────\n"
	for _, s := range services {
		res += fmt.Sprintf("%-20s %-20s %-10s\n", s.Name, s.Status, s.Uptime)
	}
	return res
}
