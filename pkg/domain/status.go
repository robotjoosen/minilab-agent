package domain

import "strings"

type Status int

const (
	StateActive Status = iota
	StateInactive
	StateIdle
	StateUnknown
)

func (s Status) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateInactive:
		return "inactive"
	case StateIdle:
		return "idle"
	default:
		return "unknown"
	}
}

func (s Status) Parse(v string) Status {
	switch strings.ToLower(v) {
	case "active", "running":
		return StateActive
	case "inactive", "exited", "dead", "created", "removing", "stopped":
		return StateInactive
	case "idle", "paused", "restarting":
		return StateIdle
	default:
		return StateUnknown
	}
}
