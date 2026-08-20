package systemd

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/robotjoosen/minilab-agent/internal/systemd/exec"
)

var execStartPathPattern = regexp.MustCompile(`path=(\S+)`)

type SystemD struct {
	runner *exec.Client
}

func New() SystemD {
	return SystemD{
		runner: exec.New(),
	}
}

func (s SystemD) GetProcesses() (Services, error) {
	out, err := s.runner.Run("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--plain")
	if err != nil {
		return nil, err
	}

	var names []string
	var states []Status
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		state := StateUnknown
		if fields[2] == StateActive.String() {
			state = StateActive
		}

		names = append(names, fields[0])
		states = append(states, state)
	}

	execStarts := s.execStarts(names)

	services := make(Services, 0, len(names))
	for i, name := range names {
		var version string
		if v, verr := s.versionFromBinary(execStarts[i]); verr == nil {
			version = v
		}

		services = append(services, Service{
			Name:    name,
			Type:    "systemd",
			State:   states[i].String(),
			Version: version,
		})
	}

	return services, nil
}

// execStarts resolves each unit's ExecStart binary path with a single
// systemctl invocation covering all units, rather than one invocation per
// unit. A host can have 80-150 units, and forking systemctl per unit adds
// up to several seconds of latency serving /capabilities and /metrics.
func (s SystemD) execStarts(names []string) []string {
	paths := make([]string, len(names))
	if len(names) == 0 {
		return paths
	}

	args := append([]string{"show"}, names...)
	args = append(args, "--property=ExecStart", "--value")

	out, err := s.runner.Run("systemctl", args...)
	if err != nil {
		return paths
	}

	lines := strings.Split(out, "\n")
	for i := range names {
		if i >= len(lines) {
			break
		}
		if m := execStartPathPattern.FindStringSubmatch(lines[i]); len(m) == 2 {
			paths[i] = m[1]
		}
	}

	return paths
}

func (s SystemD) versionFromBinary(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	return info.ModTime().UTC().Format(time.RFC3339), nil
}
