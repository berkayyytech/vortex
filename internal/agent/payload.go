package agent

import (
	"main/internal/docker"
	"main/internal/network"
	"main/internal/services"
	"main/internal/stats"
)

// Payload represents the combined JSON state returned by the remote vortex-agent.
type Payload struct {
	Stats    stats.SystemStats   `json:"stats"`
	Network  network.NetworkInfo `json:"network"`
	Docker   docker.DockerStats  `json:"docker"`
	Services []services.Service  `json:"services"`
	Logs     string              `json:"logs,omitempty"`
	Files    string              `json:"files,omitempty"`
}
