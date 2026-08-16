package discovery

import "github.com/robotjoosen/minilab-agent/internal/docker"

type Container = docker.Container

type DockerClient interface {
	ListContainers() ([]Container, error)
}
