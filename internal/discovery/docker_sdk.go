package discovery

import (
	"context"
	"strings"

	"github.com/moby/moby/client"
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
	containers, err := d.cli.ContainerList(context.Background(), client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := make([]Container, 0, len(containers.Items))
	for _, c := range containers.Items {
		name := strings.TrimPrefix(firstOrEmpty(c.Names), "/")
		result = append(result, Container{Name: name, Image: c.Image, State: string(c.State)})
	}

	return result, nil
}

func firstOrEmpty(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}
