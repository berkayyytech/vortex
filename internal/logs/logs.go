package logs

import "strings"

func GetRecentLogs() string {
	// In production, tail /var/log/syslog or application logs dynamically
	mockLogs := []string{
		"Jul 10 20:51:00 vps-manager systemd[1]: Started VPS Manager.",
		"Jul 10 20:51:05 vps-manager dockerd[431]: API listen on /var/run/docker.sock",
		"Jul 10 20:55:12 vps-manager sshd[124]: Accepted publickey for root from 192.168.1.5",
		"Jul 10 21:00:45 vps-manager nginx[512]: 192.168.1.10 - - [10/Jul/2026:21:00:45 +0000] \"GET / HTTP/1.1\" 200",
		"Jul 10 21:10:02 vps-manager kernel: [ 1234.5678] Firewall: Blocked incoming connection on port 23",
	}
	return strings.Join(mockLogs, "\n")
}
