package domain

import (
	"fmt"
	"sort"
	"strings"
)

func FormatMetrics(host HostStats, services []Service) string {
	var b strings.Builder

	fmt.Fprintf(&b, "minilab_host_cpu_percent{mode=\"user\"} %g\n", host.CPUUser)
	fmt.Fprintf(&b, "minilab_host_cpu_percent{mode=\"system\"} %g\n", host.CPUSystem)
	fmt.Fprintf(&b, "minilab_host_cpu_percent{mode=\"idle\"} %g\n", host.CPUIdle)
	fmt.Fprintf(&b, "minilab_host_memory_bytes{state=\"used\"} %d\n", host.MemUsed)
	fmt.Fprintf(&b, "minilab_host_memory_bytes{state=\"free\"} %d\n", host.MemFree)
	fmt.Fprintf(&b, "minilab_host_memory_bytes{state=\"total\"} %d\n", host.MemTotal)

	sorted := make([]Service, len(services))
	copy(sorted, services)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	for _, s := range sorted {
		up := 0
		if s.Up {
			up = 1
		}
		fmt.Fprintf(&b, "minilab_service_up{name=%q,type=%q,version=%q} %d\n", s.Name, s.Type, s.Version, up)
	}

	return b.String()
}
