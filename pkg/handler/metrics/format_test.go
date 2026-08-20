package metrics

import (
	"testing"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

func TestFormat(t *testing.T) {
	host := domain.HostStats{
		CPUUser:   12.4,
		CPUSystem: 3.1,
		CPUIdle:   84.5,
		MemUsed:   1893000000,
		MemFree:   500000000,
		MemTotal:  2393000000,
		Temperatures: []domain.Temperature{
			{Name: "cpu-thermal", Celsius: 45.3},
		},
	}
	services := domain.Services{
		{Name: "nodered.service", Type: "systemd", State: domain.StateActive, Version: "2026-08-01T10:00:00Z"},
		{Name: "ollama", Type: "docker", State: domain.StateActive, Version: "0.4.2"},
	}

	got := format(host, services)

	want := `minilab_host_cpu_percent{mode="user"} 12.4
minilab_host_cpu_percent{mode="system"} 3.1
minilab_host_cpu_percent{mode="idle"} 84.5
minilab_host_memory_bytes{state="used"} 1893000000
minilab_host_memory_bytes{state="free"} 500000000
minilab_host_memory_bytes{state="total"} 2393000000
minilab_host_temperature_celsius{sensor="cpu-thermal"} 45.3
minilab_service_up{name="nodered.service",type="systemd"} 1
minilab_service_up{name="ollama",type="docker"} 1
minilab_service_info{name="nodered.service",type="systemd",version="2026-08-01T10:00:00Z"} 1
minilab_service_info{name="ollama",type="docker",version="0.4.2"} 1
`

	if got != want {
		t.Fatalf("format() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSortsServicesByName(t *testing.T) {
	host := domain.HostStats{}
	services := domain.Services{
		{Name: "zzz.service", Type: "systemd", State: domain.StateInactive},
		{Name: "aaa.service", Type: "systemd", State: domain.StateActive},
	}

	got := format(host, services)

	aaaIdx := indexOf(got, `name="aaa.service"`)
	zzzIdx := indexOf(got, `name="zzz.service"`)
	if aaaIdx == -1 || zzzIdx == -1 || aaaIdx > zzzIdx {
		t.Fatalf("expected aaa.service before zzz.service, got:\n%s", got)
	}
}

func TestFormatSortsTemperaturesByName(t *testing.T) {
	host := domain.HostStats{
		Temperatures: []domain.Temperature{
			{Name: "zzz-thermal", Celsius: 30},
			{Name: "aaa-thermal", Celsius: 40},
		},
	}

	got := format(host, nil)

	aaaIdx := indexOf(got, `sensor="aaa-thermal"`)
	zzzIdx := indexOf(got, `sensor="zzz-thermal"`)
	if aaaIdx == -1 || zzzIdx == -1 || aaaIdx > zzzIdx {
		t.Fatalf("expected aaa-thermal before zzz-thermal, got:\n%s", got)
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
