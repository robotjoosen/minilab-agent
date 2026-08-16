package domain

type HostStats struct {
	CPUUser   float64
	CPUSystem float64
	CPUIdle   float64
	MemUsed   uint64
	MemFree   uint64
	MemTotal  uint64
}

type Service struct {
	Name    string
	Type    string // "systemd" or "docker"
	Up      bool
	Version string
}
