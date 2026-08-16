package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/httpapi"
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

func TestCapabilitiesEndpoint(t *testing.T) {
	srv := &httpapi.Server{
		Discoverer: fakeDiscoverer{services: []domain.Service{{Name: "ollama", Type: "docker", Up: true, Version: "0.4.2"}}},
		HostStats:  fakeHostStats{},
		Hostname:   "rocket",
	}

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body struct {
		Device   string          `json:"device"`
		Services []domain.Service `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Device != "rocket" || len(body.Services) != 1 || body.Services[0].Name != "ollama" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv := &httpapi.Server{
		Discoverer: fakeDiscoverer{services: []domain.Service{{Name: "ollama", Type: "docker", Up: true, Version: "0.4.2"}}},
		HostStats:  fakeHostStats{stats: domain.HostStats{CPUUser: 10}},
		Hostname:   "rocket",
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), `minilab_service_up{name="ollama",type="docker",version="0.4.2"} 1`) {
		t.Fatalf("unexpected metrics body: %s", rec.Body.String())
	}
}

func TestCapabilitiesEndpointDiscoveryError(t *testing.T) {
	srv := &httpapi.Server{
		Discoverer: fakeDiscoverer{err: errTest},
		HostStats:  fakeHostStats{},
	}

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

var errTest = &testError{"discovery failed"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
