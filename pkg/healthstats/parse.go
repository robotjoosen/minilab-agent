package healthstats

import (
	"encoding/json"

	healthdomain "github.com/robotjoosen/go-health-service/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

// ParseHealthMessage decodes the JSON payload published by go-health-service's health.ping
// messages (health-service's own exported healthdomain.SysUsageMessage shape) into this
// agent's domain.HostStats.
func ParseHealthMessage(raw []byte) (domain.HostStats, error) {
	var msg healthdomain.SysUsageMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return domain.HostStats{}, err
	}

	return domain.HostStats{
		CPUUser:   msg.Cpu.User,
		CPUSystem: msg.Cpu.System,
		CPUIdle:   msg.Cpu.Idle,
		MemUsed:   msg.Mem.Used,
		MemFree:   msg.Mem.Free,
		MemTotal:  msg.Mem.Total,
	}, nil
}
