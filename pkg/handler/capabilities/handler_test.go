package capabilities_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/handler/capabilities"
	"github.com/robotjoosen/minilab-agent/pkg/server"
)

type fakeDiscoverer struct {
	services domain.Services
	err      error
}

func (f fakeDiscoverer) Discover() (domain.Services, error) {
	return f.services, f.err
}

func TestHandleReturnsCapabilities(t *testing.T) {
	h := &capabilities.Handler{
		Discoverer: fakeDiscoverer{services: domain.Services{{Name: "ollama", Type: "docker", State: domain.StateActive, Version: "0.4.2"}}},
		Hostname:   "rocket",
	}

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	h.Handle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}

	var body struct {
		Device   string          `json:"device"`
		Services domain.Services `json:"services"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Device != "rocket" || len(body.Services) != 1 || body.Services[0].Name != "ollama" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestHandleDiscoveryErrorProducesRFC9457Problem(t *testing.T) {
	h := &capabilities.Handler{
		Discoverer: fakeDiscoverer{err: errors.New("discovery failed")},
		Hostname:   "rocket",
	}

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
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
		Instance: "/capabilities",
	}
	if problem != want {
		t.Fatalf("unexpected problem body: %+v, want %+v", problem, want)
	}
}
