package discovery_test

import (
	"errors"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/discovery"
)

type failingRunner struct {
	err error
}

func (f failingRunner) Run(name string, args ...string) (string, error) {
	return "", f.err
}

type failingDockerClient struct {
	err error
}

func (f failingDockerClient) ListContainers() ([]discovery.Container, error) {
	return nil, f.err
}

func TestAggregatorDiscoverCombinesSystemdAndDocker(t *testing.T) {
	runner := fakeRunner{outputs: map[string]string{
		"systemctl list-units --type=service --all --no-legend --plain": "nodered.service loaded active running Node-RED\n",
		"systemctl show nodered.service --property=ExecStart --value":   "{ path=/nonexistent/binary ; }",
	}}
	docker := fakeDockerClient{containers: []discovery.Container{
		{Name: "ollama", Image: "ollama/ollama:0.4.2", State: "running"},
		{Name: "rabbitmq", Image: "rabbitmq:3-management", State: "exited"},
	}}

	agg := discovery.Aggregator{Systemd: runner, Docker: docker}

	services, err := agg.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(services) != 3 {
		t.Fatalf("expected 3 services, got %d: %+v", len(services), services)
	}

	byName := map[string]bool{}
	for _, s := range services {
		byName[s.Name] = true
		if s.Name == "ollama" && (s.Type != "docker" || !s.Up || s.Version != "ollama/ollama:0.4.2") {
			t.Fatalf("unexpected ollama service: %+v", s)
		}
		if s.Name == "rabbitmq" && s.Up {
			t.Fatalf("expected rabbitmq to be reported as down (state=exited): %+v", s)
		}
		if s.Name == "nodered.service" && (s.Type != "systemd" || !s.Up) {
			t.Fatalf("unexpected nodered service: %+v", s)
		}
	}

	for _, want := range []string{"nodered.service", "ollama", "rabbitmq"} {
		if !byName[want] {
			t.Fatalf("missing expected service %q in %+v", want, services)
		}
	}
}

func TestAggregatorDiscoverToleratesDockerFailure(t *testing.T) {
	runner := fakeRunner{outputs: map[string]string{
		"systemctl list-units --type=service --all --no-legend --plain": "nodered.service loaded active running Node-RED\n",
		"systemctl show nodered.service --property=ExecStart --value":   "{ path=/nonexistent/binary ; }",
	}}
	docker := failingDockerClient{err: errors.New("docker daemon not running")}

	agg := discovery.Aggregator{Systemd: runner, Docker: docker}

	services, err := agg.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil (systemd succeeded)", err)
	}

	if len(services) != 1 || services[0].Name != "nodered.service" {
		t.Fatalf("expected only the systemd service to be reported, got %+v", services)
	}
}

func TestAggregatorDiscoverToleratesSystemdFailure(t *testing.T) {
	runner := failingRunner{err: errors.New("systemctl: command not found")}
	docker := fakeDockerClient{containers: []discovery.Container{
		{Name: "ollama", Image: "ollama/ollama:0.4.2", State: "running"},
	}}

	agg := discovery.Aggregator{Systemd: runner, Docker: docker}

	services, err := agg.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v, want nil (docker succeeded)", err)
	}

	if len(services) != 1 || services[0].Name != "ollama" {
		t.Fatalf("expected only the docker service to be reported, got %+v", services)
	}
}

func TestAggregatorDiscoverReturnsErrorWhenBothFail(t *testing.T) {
	runner := failingRunner{err: errors.New("systemctl: command not found")}
	docker := failingDockerClient{err: errors.New("docker daemon not running")}

	agg := discovery.Aggregator{Systemd: runner, Docker: docker}

	services, err := agg.Discover()
	if err == nil {
		t.Fatalf("Discover() error = nil, want error when both legs fail")
	}
	if services != nil {
		t.Fatalf("expected nil services when both legs fail, got %+v", services)
	}
}
