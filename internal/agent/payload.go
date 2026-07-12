package agent

import (
	"main/internal/docker"
	"main/internal/network"
	"main/internal/services"
	"main/internal/stats"
	"time"
)

// Payload represents the combined JSON state returned by the remote vortex-agent.
type Payload struct {
	Stats    stats.SystemStats   `json:"stats"`
	Network  network.NetworkInfo `json:"network"` // Now includes extended metrics (bandwidth, ports, etc.)
	Docker   docker.DockerStats  `json:"docker"`
	Services []services.Service  `json:"services"`
	Logs     string              `json:"logs,omitempty"`
	Files    string              `json:"files,omitempty"`
}

// PayloadErrorMsg is returned when the system engine fails to fetch the payload
type PayloadErrorMsg struct {
	Err error
}

// TickMsg signals the system engine to poll for the next payload
type TickMsg time.Time
