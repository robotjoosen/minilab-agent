package systemd

type Services []Service

type Service struct {
	Name    string
	Type    string
	State   string
	Version string
}

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
