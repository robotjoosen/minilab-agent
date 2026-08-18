package domain

type HostStats struct {
	CPUUser   float64
	CPUSystem float64
	CPUIdle   float64
	MemUsed   uint64
	MemFree   uint64
	MemTotal  uint64
}

type Services []ServiceItem

type ServiceItem struct {
	Name    string
	Type    Type
	State   Status
	Version string
}
