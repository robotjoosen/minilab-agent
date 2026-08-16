package discovery

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type SDKDockerClient struct {
	cli *client.Client
}

func NewSDKDockerClient() (*SDKDockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &SDKDockerClient{cli: cli}, nil
}

func (d *SDKDockerClient) ListContainers() ([]Container, error) {
	containers, err := d.cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(containers))
	for _, c := range containers {
		name := strings.TrimPrefix(firstOrEmpty(c.Names), "/")
		result = append(result, Container{Name: name, Image: c.Image, State: c.State})
	}

	return result, nil
}

func firstOrEmpty(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}
