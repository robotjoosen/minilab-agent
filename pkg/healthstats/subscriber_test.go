package healthstats

import (
	"reflect"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/wagslane/go-rabbitmq"
)

func healthPingFor(host string) []byte {
	return []byte(`{
		"name": "` + host + `",
		"memory": {"free": 500000000, "used": 1893000000, "total": 2393000000},
		"cpu": {"system": 3.1, "idle": 84.5, "user": 12.4},
		"network_interfaces": [],
		"disks": []
	}`)
}

func TestHandleMessageUpdatesStoreWhenHostMatches(t *testing.T) {
	store := &Store{}

	got := handleMessage(healthPingFor("beanie"), "beanie", store)

	if got != rabbitmq.Ack {
		t.Fatalf("handleMessage() = %v, want Ack", got)
	}
	if latest := store.Latest(); latest.MemTotal != 2393000000 {
		t.Fatalf("expected store to be updated, got %+v", latest)
	}
}

func TestHandleMessageSkipsStoreWhenHostDoesNotMatch(t *testing.T) {
	store := &Store{}

	got := handleMessage(healthPingFor("rocket"), "beanie", store)

	if got != rabbitmq.Ack {
		t.Fatalf("handleMessage() = %v, want Ack", got)
	}
	if latest := store.Latest(); !reflect.DeepEqual(latest, domain.HostStats{}) {
		t.Fatalf("expected store to be left untouched, got %+v", latest)
	}
}

func TestHandleMessageDiscardsInvalidJSON(t *testing.T) {
	store := &Store{}

	got := handleMessage([]byte(`not json`), "beanie", store)

	if got != rabbitmq.NackDiscard {
		t.Fatalf("handleMessage() = %v, want NackDiscard", got)
	}
	if latest := store.Latest(); !reflect.DeepEqual(latest, domain.HostStats{}) {
		t.Fatalf("expected store to be left untouched, got %+v", latest)
	}
}
