package docker

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// listTimeout bounds how long ListContainers waits on the Docker daemon.
// Hosts with no Docker daemon at all (systemd-only agents) would otherwise
// stall the caller for however long the underlying client takes to give up,
// which can be long enough to blow through a caller's own timeout.
const listTimeout = 2 * time.Second

type Container struct {
	Name  string
	Image string
	State string // "running", "exited", etc.
}

type Client struct {
	cli *client.Client
}

func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{cli: cli}, nil
}

func (c *Client) ListContainers() ([]Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listTimeout)
	defer cancel()

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(containers))
	for _, item := range containers {
		name := strings.TrimPrefix(firstOrEmpty(item.Names), "/")
		result = append(result, Container{
			Name:  name,
			Image: item.Image,
			State: item.State,
		})
	}

	return result, nil
}

func firstOrEmpty(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}
