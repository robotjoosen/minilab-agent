package discovery_test

import (
	"testing"

	"github.com/robotjoosen/minilab-agent/internal/discovery"
)

type fakeDockerClient struct {
	containers []discovery.Container
}

func (f fakeDockerClient) ListContainers() ([]discovery.Container, error) {
	return f.containers, nil
}

func TestDockerClientInterfaceIsSatisfiedByFake(t *testing.T) {
	var _ discovery.DockerClient = fakeDockerClient{}

	client := fakeDockerClient{containers: []discovery.Container{
		{Name: "ollama", Image: "ollama/ollama:0.4.2", State: "running"},
	}}

	got, err := client.ListContainers()
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "ollama" {
		t.Fatalf("unexpected containers: %+v", got)
	}
}
