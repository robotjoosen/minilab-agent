package metrics_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/handler/metrics"
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
	h := &metrics.Handler{
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
	h := &metrics.Handler{
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
