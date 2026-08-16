package healthstats

import (
	"sync"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

type Store struct {
	mu   sync.RWMutex
	last domain.HostStats
}

func (s *Store) Latest() domain.HostStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.last
}

func (s *Store) Update(h domain.HostStats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.last = h
}
