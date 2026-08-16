package discovery

import (
	"sync"
	"time"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

// CacheTTL is how long a Discover() result is reused before the underlying
// discoverer is invoked again. Discovery shells out to systemctl once per
// unit (commonly 80-150 units on a Pi) and hits the Docker API, so without a
// cache every /capabilities or /metrics HTTP request forks a large number of
// processes.
const CacheTTL = 5 * time.Second

// Discoverer is the interface CachingDiscoverer wraps. It matches the
// ServiceDiscoverer interfaces in pkg/handler/capabilities and
// pkg/handler/metrics, so *Aggregator and *CachingDiscoverer are
// interchangeable there.
type Discoverer interface {
	Discover() ([]domain.Service, error)
}

// CachingDiscoverer memoizes the last successful Discover() result for TTL,
// serving repeated calls within that window from the cache instead of
// re-invoking the wrapped Discoverer.
type CachingDiscoverer struct {
	Discoverer Discoverer
	TTL        time.Duration

	// Now returns the current time. It defaults to time.Now and exists so
	// tests can control the clock without sleeping through the TTL.
	Now func() time.Time

	mu       sync.Mutex
	services []domain.Service
	expiry   time.Time
	valid    bool
}

// NewCachingDiscoverer wraps d, caching successful results for ttl.
func NewCachingDiscoverer(d Discoverer, ttl time.Duration) *CachingDiscoverer {
	return &CachingDiscoverer{Discoverer: d, TTL: ttl, Now: time.Now}
}

func (c *CachingDiscoverer) Discover() ([]domain.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.Now
	if now == nil {
		now = time.Now
	}
	nowT := now()

	if c.valid && nowT.Before(c.expiry) {
		return c.services, nil
	}

	services, err := c.Discoverer.Discover()
	if err != nil {
		return nil, err
	}

	c.services = services
	c.expiry = nowT.Add(c.TTL)
	c.valid = true

	return c.services, nil
}
