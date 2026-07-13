package security

import (
	"fmt"
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
	UFWStatus           string
	RootLoginEnabled    bool
	PasswordAuthEnabled bool
}

type FirewallRule struct {
	ID     string
	To     string
	Action string
	From   string
}

type OpenPort struct {
	Protocol string
	Address  string
	Port     string
	Process  string
}

type FailedLogin struct {
	IP       string
	Count    int
	LastTime string
}

type Fail2BanJail struct {
	Name      string
	BannedIPs []string
}

type FullAuditReport struct {
	BasicReport  *SecurityReport
	FirewallType string
	Rules        []FirewallRule
	Ports        []OpenPort
	Logins       []FailedLogin
	Jails        []Fail2BanJail
}

// RunAudit performs a quick audit of the server's security posture
func (e *Engine) RunAudit() (*SecurityReport, error) {
	report := &SecurityReport{}

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

	sshdOut, _ := e.client.Run("cat /etc/ssh/sshd_config")
	if strings.Contains(sshdOut, "PermitRootLogin yes") {
		report.RootLoginEnabled = true
	} else {
		report.RootLoginEnabled = false
	}

	if strings.Contains(sshdOut, "PasswordAuthentication yes") {
		report.PasswordAuthEnabled = true
	} else if strings.Contains(sshdOut, "PasswordAuthentication no") {
		report.PasswordAuthEnabled = false
	} else {
		report.PasswordAuthEnabled = true
	}

	return report, nil
}

func (e *Engine) RunFullAudit() (*FullAuditReport, error) {
	report := &FullAuditReport{}
	basic, _ := e.e_RunBasic()
	report.BasicReport = basic

	fwType, rules, _ := e.GetFirewallStatus()
	report.FirewallType = fwType
	report.Rules = rules

	ports, _ := e.GetOpenPorts()
	report.Ports = ports

	logins, _ := e.GetFailedLogins()
	report.Logins = logins

	jails, _ := e.GetFail2BanStatus()
	report.Jails = jails

	return report, nil
}

func (e *Engine) e_RunBasic() (*SecurityReport, error) {
	return e.RunAudit()
}

func (e *Engine) GetFirewallStatus() (string, []FirewallRule, error) {
	out, _ := e.client.Run("which ufw")
	if strings.TrimSpace(out) != "" {
		out, _ = e.client.Run("sudo ufw status numbered")
		if strings.Contains(out, "Status: active") {
			var rules []FirewallRule
			lines := strings.Split(out, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "[") {
					parts := strings.Fields(line)
					if len(parts) >= 5 {
						id := strings.TrimRight(strings.TrimLeft(parts[0]+parts[1], "["), "]")
						
						actionIndex := -1
						for i, p := range parts {
							if p == "ALLOW" || p == "DENY" || p == "REJECT" || p == "LIMIT" {
								if actionIndex == -1 {
									actionIndex = i
								}
							}
						}
						
						if actionIndex > 1 {
							to := strings.Join(parts[1:actionIndex], " ")
							if id != "" && !strings.Contains(to, "]") {
								action := parts[actionIndex]
								from := ""
								if len(parts) > actionIndex+1 {
									if parts[actionIndex+1] == "IN" || parts[actionIndex+1] == "OUT" {
										action += " " + parts[actionIndex+1]
										if len(parts) > actionIndex+2 {
											from = strings.Join(parts[actionIndex+2:], " ")
										}
									} else {
										from = strings.Join(parts[actionIndex+1:], " ")
									}
								}
								rules = append(rules, FirewallRule{ID: id, To: to, Action: action, From: from})
							}
						}
					}
				}
			}
			return "ufw", rules, nil
		}
		return "ufw (inactive)", nil, nil
	}

	out, _ = e.client.Run("sudo iptables -L INPUT -n --line-numbers")
	var rules []FirewallRule
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] != "num" && fields[0] != "Chain" {
			rules = append(rules, FirewallRule{
				ID:     fields[0],
				Action: fields[1],
				To:     fields[4], // destination
				From:   fields[3], // source
			})
		}
	}
	if len(rules) > 0 {
		return "iptables", rules, nil
	}
	return "none", nil, nil
}

func (e *Engine) AddFirewallRule(port, proto, action string) error {
	fwType, _, _ := e.GetFirewallStatus()
	if strings.Contains(fwType, "ufw") {
		cmd := fmt.Sprintf("sudo ufw %s %s/%s", strings.ToLower(action), port, proto)
		if proto == "any" {
			cmd = fmt.Sprintf("sudo ufw %s %s", strings.ToLower(action), port)
		}
		_, err := e.client.Run(cmd)
		return err
	} else if fwType == "iptables" {
		target := "ACCEPT"
		if strings.ToLower(action) == "deny" {
			target = "DROP"
		}
		cmd := fmt.Sprintf("sudo iptables -A INPUT -p %s --dport %s -j %s", proto, port, target)
		if proto == "any" {
			cmd = fmt.Sprintf("sudo iptables -A INPUT -p tcp --dport %s -j %s && sudo iptables -A INPUT -p udp --dport %s -j %s", port, target, port, target)
		}
		_, err := e.client.Run(cmd)
		return err
	}
	return fmt.Errorf("no supported firewall active")
}

func (e *Engine) RemoveFirewallRule(id string) error {
	fwType, _, _ := e.GetFirewallStatus()
	if strings.Contains(fwType, "ufw") {
		cmd := fmt.Sprintf("sudo ufw --force delete %s", id)
		_, err := e.client.Run(cmd)
		return err
	} else if fwType == "iptables" {
		cmd := fmt.Sprintf("sudo iptables -D INPUT %s", id)
		_, err := e.client.Run(cmd)
		return err
	}
	return fmt.Errorf("no supported firewall active")
}

func (e *Engine) GetOpenPorts() ([]OpenPort, error) {
	out, err := e.client.Run("sudo ss -tulpn")
	if err != nil {
		out, err = e.client.Run("sudo netstat -tulpn")
		if err != nil {
			return nil, err
		}
	}

	var ports []OpenPort
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(line, "LISTEN") {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				var proto, local, proc string
				if strings.HasPrefix(fields[0], "tcp") || strings.HasPrefix(fields[0], "udp") {
					if strings.Contains(line, "Local Address") {
						continue
					}
					proto = fields[0]
					local = fields[3]
					proc = fields[len(fields)-1]
				} else if fields[1] == "LISTEN" || fields[1] == "UNCONN" {
					proto = fields[0]
					local = fields[4]
					proc = strings.Join(fields[6:], " ")
				} else {
					continue
				}

				lastColon := strings.LastIndex(local, ":")
				if lastColon != -1 {
					addr := local[:lastColon]
					port := local[lastColon+1:]
					ports = append(ports, OpenPort{
						Protocol: proto,
						Address:  addr,
						Port:     port,
						Process:  proc,
					})
				}
			}
		}
	}
	return ports, nil
}

func (e *Engine) GetFailedLogins() ([]FailedLogin, error) {
	cmd := `journalctl -u ssh --no-pager --since "10 days ago" | grep "Failed password" | tail -n 1000`
	out, err := e.client.Run(cmd)
	if err != nil {
		cmd = `grep "Failed password" /var/log/auth.log | tail -n 1000`
		out, err = e.client.Run(cmd)
		if err != nil {
			return nil, err
		}
	}

	counts := make(map[string]int)
	lastTimes := make(map[string]string)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		idx := strings.Index(line, "from ")
		if idx != -1 {
			parts := strings.Fields(line[idx+5:])
			if len(parts) > 0 {
				ip := parts[0]
				counts[ip]++

				timestamp := ""
				if len(line) >= 15 {
					if line[3] == ' ' && line[6] == ' ' {
						timestamp = line[:15]
					} else {
						timestamp = strings.Fields(line)[0]
						if len(timestamp) > 10 && timestamp[10] == 'T' {
							// ISO8601
						} else {
							// Just grab first 3 words as fallback
							fields := strings.Fields(line)
							if len(fields) >= 3 {
								timestamp = fields[0] + " " + fields[1] + " " + fields[2]
							}
						}
					}
				}
				lastTimes[ip] = timestamp
			}
		}
	}

	var logins []FailedLogin
	for ip, count := range counts {
		logins = append(logins, FailedLogin{
			IP:       ip,
			Count:    count,
			LastTime: lastTimes[ip],
		})
	}
	return logins, nil
}

func (e *Engine) GetFail2BanStatus() ([]Fail2BanJail, error) {
	out, err := e.client.Run("sudo fail2ban-client status")
	if err != nil {
		return nil, fmt.Errorf("fail2ban not installed or accessible")
	}

	var jails []Fail2BanJail
	lines := strings.Split(out, "\n")
	var jailList []string
	for _, line := range lines {
		if strings.Contains(line, "Jail list:") {
			parts := strings.Split(line, "Jail list:")
			if len(parts) == 2 {
				jNames := strings.Split(parts[1], ",")
				for _, j := range jNames {
					jailList = append(jailList, strings.TrimSpace(j))
				}
			}
		}
	}

	for _, jail := range jailList {
		if jail == "" {
			continue
		}
		out, _ := e.client.Run(fmt.Sprintf("sudo fail2ban-client status %s", jail))
		j := Fail2BanJail{Name: jail}
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Banned IP list:") {
				parts := strings.Split(line, "Banned IP list:")
				if len(parts) == 2 {
					ips := strings.Fields(parts[1])
					j.BannedIPs = ips
				}
			}
		}
		jails = append(jails, j)
	}
	return jails, nil
}

func (e *Engine) BanIP(jail, ip string) error {
	_, err := e.client.Run(fmt.Sprintf("sudo fail2ban-client set %s banip %s", jail, ip))
	return err
}

func (e *Engine) UnbanIP(jail, ip string) error {
	_, err := e.client.Run(fmt.Sprintf("sudo fail2ban-client set %s unbanip %s", jail, ip))
	return err
}
