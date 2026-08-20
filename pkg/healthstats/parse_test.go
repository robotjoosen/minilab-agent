package healthstats_test

import (
	"reflect"
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/healthstats"
)

func TestParseHealthMessage(t *testing.T) {
	raw := []byte(`{
		"name": "rocket",
		"memory": {"free": 500000000, "used": 1893000000, "total": 2393000000},
		"cpu": {"system": 3.1, "idle": 84.5, "user": 12.4},
		"network_interfaces": [],
		"disks": [],
		"temperatures": [{"name": "cpu-thermal", "celsius": 45.3}]
	}`)

	host, got, err := healthstats.ParseHealthMessage(raw)
	if err != nil {
		t.Fatalf("ParseHealthMessage() error = %v", err)
	}

	if host != "rocket" {
		t.Fatalf("unexpected host: %q", host)
	}
	if got.CPUUser != 12.4 || got.CPUSystem != 3.1 || got.CPUIdle != 84.5 {
		t.Fatalf("unexpected cpu stats: %+v", got)
	}
	if got.MemUsed != 1893000000 || got.MemFree != 500000000 || got.MemTotal != 2393000000 {
		t.Fatalf("unexpected memory stats: %+v", got)
	}
	want := []domain.Temperature{{Name: "cpu-thermal", Celsius: 45.3}}
	if !reflect.DeepEqual(got.Temperatures, want) {
		t.Fatalf("unexpected temperatures: %+v, want %+v", got.Temperatures, want)
	}
}

func TestParseHealthMessageInvalidJSON(t *testing.T) {
	_, _, err := healthstats.ParseHealthMessage([]byte(`not json`))
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}
