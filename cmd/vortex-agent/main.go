package main

import (
	"encoding/json"
	"fmt"
	"os"

	"main/internal/agent"
	"main/internal/docker"
	"main/internal/network"
	"main/internal/services"
	"main/internal/stats"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: vortex-agent [command]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "payload":
		// Gather all stats and output as a single JSON payload
		payload := agent.Payload{
			Stats:    stats.GetSystemStats(),
			Network:  network.GetNetworkStats(),
			Docker:   docker.GetDockerStats(),
			Services: services.GetServices(),
		}
		
		out, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf(`{"error": "%v"}`, err)
			os.Exit(1)
		}
		fmt.Println(string(out))
		
	default:
		fmt.Println("Unknown command")
		os.Exit(1)
	}
}
