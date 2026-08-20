package healthstats

import (
	"encoding/json"

	healthdomain "github.com/robotjoosen/go-health-service/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

// ParseHealthMessage decodes the JSON payload published by go-health-service's health.ping
// messages (health-service's own exported healthdomain.SysUsageMessage shape) into the
// sending host's name and this agent's domain.HostStats.
//
// The name is returned separately rather than folded into domain.HostStats because every
// agent's queue receives every host's ping (see Subscribe) -- callers need it to tell whose
// stats they're looking at before deciding whether to keep them.
func ParseHealthMessage(raw []byte) (host string, stats domain.HostStats, err error) {
	var msg healthdomain.SysUsageMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return "", domain.HostStats{}, err
	}

	return msg.Name, domain.HostStats{
		CPUUser:   msg.Cpu.User,
		CPUSystem: msg.Cpu.System,
		CPUIdle:   msg.Cpu.Idle,
		MemUsed:   msg.Mem.Used,
		MemFree:   msg.Mem.Free,
		MemTotal:  msg.Mem.Total,
	}, nil
}
