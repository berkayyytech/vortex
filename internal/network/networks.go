package network

import (
	"github.com/showwin/speedtest-go/speedtest"
)

type NetworkInfo struct {
	ServerName string
	Ping       string
	Status     string
	Download   float64
	Upload     float64
}

// GetNetworkStats fetches the nearest speedtest server info asynchronously
func GetNetworkStats() NetworkInfo {
	info := NetworkInfo{Status: "Fetching..."}
	var client = speedtest.New()
	serverList, err := client.FetchServers()
	if err != nil {
		info.Status = "Error fetching servers"
		return info
	}

	targets, err := serverList.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		info.Status = "No servers found"
		return info
	}

	s := targets[0]
	err = s.PingTest(nil)
	info.ServerName = s.Name
	if err == nil {
		info.Ping = s.Latency.String()
	} else {
		info.Ping = "N/A"
	}

	err = s.DownloadTest()
	if err == nil {
		info.Download = float64(s.DLSpeed) / 125000 // Convert Bytes/sec to Mbps
	}

	err = s.UploadTest()
	if err == nil {
		info.Upload = float64(s.ULSpeed) / 125000 // Convert Bytes/sec to Mbps
	}

	info.Status = "Complete"
	return info
}
