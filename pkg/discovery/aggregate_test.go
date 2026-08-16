package discovery_test

import (
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/discovery"
)

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
