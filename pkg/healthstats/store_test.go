package healthstats_test

import (
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/healthstats"
)

func TestStoreUpdateAndLatest(t *testing.T) {
	store := &healthstats.Store{}

	if got := store.Latest(); got != (domain.HostStats{}) {
		t.Fatalf("expected zero value before any update, got %+v", got)
	}

	want := domain.HostStats{CPUUser: 1, MemUsed: 2}
	store.Update(want)

	if got := store.Latest(); got != want {
		t.Fatalf("Latest() = %+v, want %+v", got, want)
	}
}
