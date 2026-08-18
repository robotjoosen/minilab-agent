package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

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
	containers, err := c.cli.ContainerList(context.Background(), container.ListOptions{All: true})
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
