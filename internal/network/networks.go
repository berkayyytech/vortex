package network

import (
	"bytes"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type NetworkInfo struct {
	// Original fields
	ServerName string  `json:"server_name,omitempty"`
	Ping       string  `json:"ping,omitempty"`
	Status     string  `json:"status,omitempty"`
	Download   float64 `json:"download,omitempty"`
	Upload     float64 `json:"upload,omitempty"`

	// New fields
	PublicIP   string          `json:"public_ip"`
	PrivateIP  string          `json:"private_ip"`
	Hostname   string          `json:"hostname"`
	Gateway    string          `json:"gateway"`
	Interfaces []Interface     `json:"interfaces"`
	Bandwidth  Bandwidth       `json:"bandwidth"`
	Ports      []Port          `json:"ports"`
	Connection ConnectionStats `json:"connection_stats"`
}

type Interface struct {
	Name   string `json:"name"`
	Status string `json:"status"` // UP or DOWN
	IPv4   string `json:"ipv4"`
	IPv6   string `json:"ipv6"`
	Type   string `json:"type"` // e.g., loopback, ethernet
	RxRate uint64 `json:"rx_rate"` // current bytes/sec
	TxRate uint64 `json:"tx_rate"` // current bytes/sec
}

type Bandwidth struct {
	CurrentRx uint64 `json:"current_rx"` // bytes/sec
	CurrentTx uint64 `json:"current_tx"` // bytes/sec
	TotalRx   uint64 `json:"total_rx"`   // bytes
	TotalTx   uint64 `json:"total_tx"`   // bytes
}

type Port struct {
	Protocol string `json:"protocol"` // tcp, udp, tcp6, udp6
	Address  string `json:"address"`  // local address including port
	State    string `json:"state"`    // LISTEN, ESTABLISHED, etc.
	Process  string `json:"process"`  // pid/process name
}

type ConnectionStats struct {
	ActiveTCP   int `json:"active_tcp"`
	ActiveUDP   int `json:"active_udp"`
	Established int `json:"established"`
	Errors      int `json:"errors"`
}

var (
	cachedPublicIP string
	lastNetDev     time.Time
	lastRx         = make(map[string]uint64)
	lastTx         = make(map[string]uint64)
)

func getPublicIP() string {
	if cachedPublicIP != "" && cachedPublicIP != "N/A" {
		return cachedPublicIP
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://ifconfig.me/ip")
	if err == nil {
		defer resp.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		ip := strings.TrimSpace(buf.String())
		if ip != "" {
			cachedPublicIP = ip
			return ip
		}
	}
	return "N/A"
}

func getHostname() string {
	b, err := exec.Command("hostname").Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(b))
}

func getDefaultGatewayAndPrivateIP() (string, string) {
	gateway := "N/A"
	privateIP := "N/A"
	
	out, err := exec.Command("ip", "route").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "default via") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					gateway = parts[2]
				}
				for i, p := range parts {
					if p == "dev" && i+1 < len(parts) {
						iface := parts[i+1]
						privateIP = getIfaceIP(iface)
						break
					}
				}
				if gateway != "N/A" {
					break
				}
			}
		}
	}
	return gateway, privateIP
}

func getIfaceIP(iface string) string {
	out, err := exec.Command("ip", "-4", "addr", "show", "dev", iface).Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "inet ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					return strings.Split(parts[1], "/")[0]
				}
			}
		}
	}
	return "N/A"
}

type IfaceTemp struct {
	Status string
	IPv4   []string
	IPv6   []string
	Type   string
}

func getInterfacesInfo() map[string]*IfaceTemp {
	m := make(map[string]*IfaceTemp)
	
	out, err := exec.Command("ip", "-o", "link").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if line == "" { continue }
			parts := strings.Split(line, ": ")
			if len(parts) >= 2 {
				name := strings.TrimSpace(parts[1])
				name = strings.Split(name, "@")[0]
				status := "DOWN"
				if strings.Contains(parts[2], "state UP") || strings.Contains(parts[2], "state UNKNOWN") {
					status = "UP"
				}
				typ := "ethernet"
				if strings.Contains(parts[2], "loopback") || name == "lo" {
					typ = "loopback"
				}
				m[name] = &IfaceTemp{Status: status, Type: typ}
			}
		}
	}
	
	out, err = exec.Command("ip", "-o", "addr").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if line == "" { continue }
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				name := strings.TrimSpace(fields[1])
				name = strings.Split(name, "@")[0]
				if m[name] == nil {
					m[name] = &IfaceTemp{Status: "UNKNOWN", Type: "unknown"}
				}
				if fields[2] == "inet" {
					m[name].IPv4 = append(m[name].IPv4, strings.Split(fields[3], "/")[0])
				} else if fields[2] == "inet6" {
					m[name].IPv6 = append(m[name].IPv6, strings.Split(fields[3], "/")[0])
				}
			}
		}
	}
	return m
}

func parseNetDev() map[string][2]uint64 {
	res := make(map[string][2]uint64)
	out, err := exec.Command("cat", "/proc/net/dev").Output()
	if err != nil {
		return res
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines { 
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) >= 9 {
			rx, _ := strconv.ParseUint(fields[0], 10, 64)
			tx, _ := strconv.ParseUint(fields[8], 10, 64)
			res[iface] = [2]uint64{rx, tx}
		}
	}
	return res
}

func gatherInterfacesAndBandwidth(bw *Bandwidth) []Interface {
	var ifaces []Interface
	currentDev := parseNetDev()
	now := time.Now()
	var duration float64 = 0
	if !lastNetDev.IsZero() {
		duration = now.Sub(lastNetDev).Seconds()
	}
	
	info := getInterfacesInfo()
	
	var totalRx, totalTx, currentRx, currentTx uint64
	
	for name, iface := range info {
		var rx, tx, rxRate, txRate uint64
		if dev, ok := currentDev[name]; ok {
			rx = dev[0]
			tx = dev[1]
			
			if lastRx[name] > 0 && duration > 0 && rx >= lastRx[name] {
				rxRate = uint64(float64(rx - lastRx[name]) / duration)
			}
			if lastTx[name] > 0 && duration > 0 && tx >= lastTx[name] {
				txRate = uint64(float64(tx - lastTx[name]) / duration)
			}
			lastRx[name] = rx
			lastTx[name] = tx
			
			if name != "lo" {
				totalRx += rx
				totalTx += tx
				currentRx += rxRate
				currentTx += txRate
			}
		}
		
		ipv4 := strings.Join(iface.IPv4, ", ")
		ipv6 := strings.Join(iface.IPv6, ", ")
		
		ifaces = append(ifaces, Interface{
			Name:   name,
			Status: iface.Status,
			IPv4:   ipv4,
			IPv6:   ipv6,
			Type:   iface.Type,
			RxRate: rxRate,
			TxRate: txRate,
		})
	}
	lastNetDev = now
	
	bw.CurrentRx = currentRx
	bw.CurrentTx = currentTx
	bw.TotalRx = totalRx
	bw.TotalTx = totalTx
	
	return ifaces
}

func getListeningPorts() []Port {
	var ports []Port
	out, err := exec.Command("ss", "-tulpn").Output()
	if err != nil {
		return ports
	}
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 || line == "" { continue }
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			proto := fields[0]
			state := fields[1]
			
			if state != "LISTEN" && state != "UNCONN" {
				continue
			}
			
			localAddr := fields[4]
			process := ""
			if len(fields) >= 7 {
			    process = strings.Join(fields[6:], " ")
			}
			
			ports = append(ports, Port{
				Protocol: proto,
				Address:  localAddr,
				State:    state,
				Process:  process,
			})
		}
	}
	return ports
}

func getConnectionStats() ConnectionStats {
	var stats ConnectionStats
	out, err := exec.Command("ss", "-s").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "TCP:") {
				parts := strings.Split(line, "estab ")
				if len(parts) > 1 {
					estabStr := strings.Split(parts[1], ",")[0]
					stats.Established, _ = strconv.Atoi(strings.TrimSpace(estabStr))
				}
			}
			if strings.HasPrefix(line, "UDP:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					stats.ActiveUDP, _ = strconv.Atoi(fields[1])
				}
			}
			if strings.HasPrefix(line, "TCP") && !strings.HasPrefix(line, "TCP:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					stats.ActiveTCP, _ = strconv.Atoi(fields[1])
				}
			}
		}
	}
	
	out, err = exec.Command("cat", "/proc/net/snmp").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "Udp:") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "Udp:") {
				headers := strings.Fields(line)
				values := strings.Fields(lines[i+1])
				for j, h := range headers {
					if h == "InErrors" || h == "RcvbufErrors" || h == "SndbufErrors" {
						if j < len(values) {
							v, _ := strconv.Atoi(values[j])
							stats.Errors += v
						}
					}
				}
			}
			if strings.HasPrefix(line, "Tcp:") && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "Tcp:") {
				headers := strings.Fields(line)
				values := strings.Fields(lines[i+1])
				for j, h := range headers {
					if h == "RetransSegs" || h == "InErrs" || h == "OutRsts" {
						if j < len(values) {
							v, _ := strconv.Atoi(values[j])
							stats.Errors += v
						}
					}
				}
			}
		}
	}
	return stats
}

// GetNetworkStats fetches all network metrics quickly
func GetNetworkStats() NetworkInfo {
	var stats NetworkInfo
	stats.PublicIP = getPublicIP()
	stats.Hostname = getHostname()
	stats.Gateway, stats.PrivateIP = getDefaultGatewayAndPrivateIP()
	stats.Interfaces = gatherInterfacesAndBandwidth(&stats.Bandwidth)
	stats.Ports = getListeningPorts()
	stats.Connection = getConnectionStats()
	
	stats.Status = "Complete"
	stats.ServerName = "Local System"

	return stats
}
