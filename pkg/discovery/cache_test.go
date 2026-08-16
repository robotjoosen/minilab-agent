package discovery_test

import (
	"testing"
	"time"

	"github.com/robotjoosen/minilab-agent/pkg/discovery"
	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

type countingDiscoverer struct {
	calls    int
	services []domain.Service
	err      error
}

func (c *countingDiscoverer) Discover() ([]domain.Service, error) {
	c.calls++
	return c.services, c.err
}

func TestCachingDiscovererServesFromCacheWithinTTL(t *testing.T) {
	inner := &countingDiscoverer{services: []domain.Service{{Name: "ollama", Type: "docker", Up: true}}}
	cache := discovery.NewCachingDiscoverer(inner, 5*time.Second)

	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	cache.Now = func() time.Time { return now }

	if _, err := cache.Discover(); err != nil {
		t.Fatalf("first Discover() error = %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 call to underlying discoverer, got %d", inner.calls)
	}

	// Still within the TTL window: should be served from cache.
	now = now.Add(3 * time.Second)
	services, err := cache.Discover()
	if err != nil {
		t.Fatalf("second Discover() error = %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected underlying discoverer not to be re-invoked within TTL, got %d calls", inner.calls)
	}
	if len(services) != 1 || services[0].Name != "ollama" {
		t.Fatalf("unexpected cached services: %+v", services)
	}
}

func TestCachingDiscovererRefreshesAfterTTLExpires(t *testing.T) {
	inner := &countingDiscoverer{services: []domain.Service{{Name: "ollama", Type: "docker", Up: true}}}
	cache := discovery.NewCachingDiscoverer(inner, 5*time.Second)

	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	cache.Now = func() time.Time { return now }

	if _, err := cache.Discover(); err != nil {
		t.Fatalf("first Discover() error = %v", err)
	}
	if inner.calls != 1 {
		t.Fatalf("expected 1 call to underlying discoverer, got %d", inner.calls)
	}

	// Past the TTL window: should re-invoke the underlying discoverer.
	now = now.Add(6 * time.Second)
	if _, err := cache.Discover(); err != nil {
		t.Fatalf("second Discover() error = %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("expected underlying discoverer to be re-invoked after TTL, got %d calls", inner.calls)
	}
}
