package metrics

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/server"
)

type fakeDiscoverer struct {
	services []domain.Service
	err      error
}

func (f fakeDiscoverer) Discover() ([]domain.Service, error) {
	return f.services, f.err
}

type fakeHostStats struct {
	stats domain.HostStats
}

func (f fakeHostStats) Latest() domain.HostStats {
	return f.stats
}

func TestHandleReturnsPrometheusMetrics(t *testing.T) {
	h := &Handler{
		Discoverer: fakeDiscoverer{services: []domain.Service{{Name: "ollama", Type: "docker", Up: true, Version: "0.4.2"}}},
		HostStats:  fakeHostStats{stats: domain.HostStats{CPUUser: 10}},
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; version=0.0.4" {
		t.Fatalf("expected text/plain; version=0.0.4 content type, got %q", ct)
	}

	if !strings.Contains(rec.Body.String(), `minilab_service_up{name="ollama",type="docker"} 1`) {
		t.Fatalf("unexpected metrics body: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `minilab_service_info{name="ollama",type="docker",version="0.4.2"} 1`) {
		t.Fatalf("unexpected metrics body: %s", rec.Body.String())
	}
}

func TestHandleDiscoveryErrorProducesRFC9457Problem(t *testing.T) {
	h := &Handler{
		Discoverer: fakeDiscoverer{err: errors.New("discovery failed")},
		HostStats:  fakeHostStats{},
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json content type, got %q", ct)
	}

	var problem server.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	want := server.Problem{
		Type:     "about:blank",
		Title:    "discovery failed",
		Status:   http.StatusInternalServerError,
		Detail:   "discovery failed",
		Instance: "/metrics",
	}
	if problem != want {
		t.Fatalf("unexpected problem body: %+v, want %+v", problem, want)
	}
}

func TestFormatMetrics(t *testing.T) {
	host := domain.HostStats{
		CPUUser:   12.4,
		CPUSystem: 3.1,
		CPUIdle:   84.5,
		MemUsed:   1893000000,
		MemFree:   500000000,
		MemTotal:  2393000000,
	}
	services := []domain.Service{
		{Name: "nodered.service", Type: "systemd", Up: true, Version: "2026-08-01T10:00:00Z"},
		{Name: "ollama", Type: "docker", Up: true, Version: "0.4.2"},
	}

	got := format(host, services)

	want := `minilab_host_cpu_percent{mode="user"} 12.4
minilab_host_cpu_percent{mode="system"} 3.1
minilab_host_cpu_percent{mode="idle"} 84.5
minilab_host_memory_bytes{state="used"} 1893000000
minilab_host_memory_bytes{state="free"} 500000000
minilab_host_memory_bytes{state="total"} 2393000000
minilab_service_up{name="nodered.service",type="systemd"} 1
minilab_service_up{name="ollama",type="docker"} 1
minilab_service_info{name="nodered.service",type="systemd",version="2026-08-01T10:00:00Z"} 1
minilab_service_info{name="ollama",type="docker",version="0.4.2"} 1
`

	if got != want {
		t.Fatalf("format() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatMetricsSortsServicesByName(t *testing.T) {
	host := domain.HostStats{}
	services := []domain.Service{
		{Name: "zzz.service", Type: "systemd", Up: false},
		{Name: "aaa.service", Type: "systemd", Up: true},
	}

	got := format(host, services)

	aaaIdx := indexOf(got, `name="aaa.service"`)
	zzzIdx := indexOf(got, `name="zzz.service"`)
	if aaaIdx == -1 || zzzIdx == -1 || aaaIdx > zzzIdx {
		t.Fatalf("expected aaa.service before zzz.service, got:\n%s", got)
	}
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
