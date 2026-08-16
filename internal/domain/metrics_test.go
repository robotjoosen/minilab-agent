package domain_test

import (
	"testing"

	"github.com/robotjoosen/minilab-agent/internal/domain"
)

func TestFormatMetrics(t *testing.T) {
	host := domain.HostStats{
		CPUUser:   12.4,
		CPUSystem: 3.1,
		CPUIdle:   84.5,
		MemUsed:   1893000000,
		MemFree:   500000000,
		MemTotal:  2393000000,
	}
	services := []domain.Service{
		{Name: "nodered.service", Type: "systemd", Up: true, Version: "2026-08-01T10:00:00Z"},
		{Name: "ollama", Type: "docker", Up: true, Version: "0.4.2"},
	}

	got := domain.FormatMetrics(host, services)

	want := `minilab_host_cpu_percent{mode="user"} 12.4
minilab_host_cpu_percent{mode="system"} 3.1
minilab_host_cpu_percent{mode="idle"} 84.5
minilab_host_memory_bytes{state="used"} 1893000000
minilab_host_memory_bytes{state="free"} 500000000
minilab_host_memory_bytes{state="total"} 2393000000
minilab_service_up{name="nodered.service",type="systemd",version="2026-08-01T10:00:00Z"} 1
minilab_service_up{name="ollama",type="docker",version="0.4.2"} 1
`

	if got != want {
		t.Fatalf("FormatMetrics() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatMetricsSortsServicesByName(t *testing.T) {
	host := domain.HostStats{}
	services := []domain.Service{
		{Name: "zzz.service", Type: "systemd", Up: false},
		{Name: "aaa.service", Type: "systemd", Up: true},
	}

	got := domain.FormatMetrics(host, services)

	aaaIdx := indexOf(got, `name="aaa.service"`)
	zzzIdx := indexOf(got, `name="zzz.service"`)
	if aaaIdx == -1 || zzzIdx == -1 || aaaIdx > zzzIdx {
		t.Fatalf("expected aaa.service before zzz.service, got:\n%s", got)
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
