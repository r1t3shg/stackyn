package dockerrun

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Runner struct {
	client *client.Client
}

func NewRunner(dockerHost string) (*Runner, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Runner{client: cli}, nil
}

func (r *Runner) Run(ctx context.Context, imageName, subdomain, baseDomain string) (string, error) {
	// Build FQDN and determine router/service names
	fqdn := fmt.Sprintf("%s.%s", subdomain, baseDomain)
	routerName := subdomain
	serviceName := subdomain
	containerName := subdomain
	internalPort := 8080 // Default port, can be made configurable if needed

	// Create Traefik labels with HTTPS/TLS support
	labels := map[string]string{
		"traefik.enable":                                                     "true",
		"traefik.docker.network":                                             "stackyn-network",
		"traefik.http.routers." + routerName + ".rule":                       fmt.Sprintf("Host(`%s`)", fqdn),
		"traefik.http.routers." + routerName + ".entrypoints":                "websecure",
		"traefik.http.routers." + routerName + ".tls":                        "true",
		"traefik.http.routers." + routerName + ".tls.certresolver":           "le",
		"traefik.http.services." + serviceName + ".loadbalancer.server.port": strconv.Itoa(internalPort),
	}

	// Create container config
	containerConfig := &container.Config{
		Image:  imageName,
		Labels: labels,
	}

	// Create host config
	hostConfig := &container.HostConfig{
		AutoRemove: false,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Create network config to connect to stackyn-network
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"stackyn-network": {},
		},
	}

	// Create container
	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := r.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

func (r *Runner) Stop(ctx context.Context, containerID string) error {
	return r.client.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (r *Runner) Remove(ctx context.Context, containerID string) error {
	return r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}
