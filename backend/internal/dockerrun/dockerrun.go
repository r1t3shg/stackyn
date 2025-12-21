// Package dockerrun provides functionality for running Docker containers.
// It handles container creation, startup, and Traefik label configuration
// for automatic routing. Containers are configured with:
//   - Traefik labels for automatic service discovery
//   - SSL/TLS support via Let's Encrypt
//   - Network configuration for Traefik routing
//   - Port mapping and health checks
package dockerrun

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Runner struct {
	client *client.Client
}

func NewRunner(dockerHost string) (*Runner, error) {
	log.Printf("[DOCKER] Initializing Docker runner - Host: %s", dockerHost)
	cli, err := client.NewClientWithOpts(
		client.WithHost(dockerHost),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Printf("[DOCKER] ERROR - Failed to create Docker client: %v", err)
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	log.Printf("[DOCKER] Docker runner initialized successfully")
	return &Runner{client: cli}, nil
}

func (r *Runner) Run(ctx context.Context, imageName, subdomain, baseDomain string) (string, error) {
	// Build FQDN and determine router/service names
	fqdn := fmt.Sprintf("%s.%s", subdomain, baseDomain)
	routerName := subdomain
	serviceName := subdomain
	containerName := subdomain
	internalPort := 8080 // Default port, can be made configurable if needed

	log.Printf("[DOCKER] Running container - Image: %s, Subdomain: %s, FQDN: %s", imageName, subdomain, fqdn)

	// Create Traefik labels with HTTPS/TLS support
	labels := map[string]string{
		"traefik.enable": "true",
		"traefik.docker.network": "stackyn-network",
		// HTTPS Router
		"traefik.http.routers." + routerName + ".rule":                       fmt.Sprintf("Host(`%s`)", fqdn),
		"traefik.http.routers." + routerName + ".entrypoints":                "websecure",
		"traefik.http.routers." + routerName + ".tls":                        "true",
		"traefik.http.routers." + routerName + ".tls.certresolver":           "letsencrypt",
		"traefik.http.routers." + routerName + ".service":                    serviceName,
		// HTTP Router (redirects to HTTPS using inline redirect middleware)
		"traefik.http.routers." + routerName + "-redirect.rule":              fmt.Sprintf("Host(`%s`)", fqdn),
		"traefik.http.routers." + routerName + "-redirect.entrypoints":       "web",
		"traefik.http.routers." + routerName + "-redirect.middlewares":       routerName + "-redirect",
		// Redirect middleware (inline)
		"traefik.http.middlewares." + routerName + "-redirect.redirectscheme.scheme": "https",
		"traefik.http.middlewares." + routerName + "-redirect.redirectscheme.permanent": "true",
		// Service definition
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
	log.Printf("[DOCKER] Creating container: %s", containerName)
	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		log.Printf("[DOCKER] ERROR - Failed to create container: %v", err)
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	log.Printf("[DOCKER] Container created - ID: %s", resp.ID)

	// Start container
	log.Printf("[DOCKER] Starting container: %s", resp.ID)
	if err := r.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Printf("[DOCKER] ERROR - Failed to start container: %v", err)
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("[DOCKER] Container started successfully - ID: %s, Name: %s, URL: https://%s", resp.ID, containerName, fqdn)
	return resp.ID, nil
}

func (r *Runner) Stop(ctx context.Context, containerID string) error {
	return r.client.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (r *Runner) Remove(ctx context.Context, containerID string) error {
	return r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}
