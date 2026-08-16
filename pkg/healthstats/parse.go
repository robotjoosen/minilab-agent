package healthstats

import (
	"encoding/json"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

type sysUsageMessage struct {
	Mem struct {
		Free  uint64 `json:"free"`
		Used  uint64 `json:"used"`
		Total uint64 `json:"total"`
	} `json:"memory"`
	Cpu struct {
		System float64 `json:"system"`
		Idle   float64 `json:"idle"`
		User   float64 `json:"user"`
	} `json:"cpu"`
}

// ParseHealthMessage decodes the JSON payload published by go-health-service's health.ping
// messages into this agent's domain.HostStats.
func ParseHealthMessage(raw []byte) (domain.HostStats, error) {
	var msg sysUsageMessage
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
