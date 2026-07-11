package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type DockerStats struct {
	Containers int
	Images     int
	Networks   int
	Volumes    int
	Status     string
}

// GetDockerStats fetches the current metrics from the Docker daemon
func GetDockerStats() DockerStats {
	stats := DockerStats{Status: "Connected"}

	// Connect to the local Docker daemon
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		stats.Status = fmt.Sprintf("Error connecting: %v", err)
		return stats
	}
	defer cli.Close()

	ctx := context.Background()

	// Containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err == nil {
		stats.Containers = len(containers)
	} else {
		stats.Status = fmt.Sprintf("Error reading containers: %v", err)
	}

	// Images
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err == nil {
		stats.Images = len(images)
	}

	// Networks
	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err == nil {
		stats.Networks = len(networks)
	}

	// Volumes
	vols, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err == nil {
		stats.Volumes = len(vols.Volumes)
	}

	return stats
}
