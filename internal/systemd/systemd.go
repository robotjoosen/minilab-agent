package systemd

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/robotjoosen/minilab-agent/internal/systemd/exec"
)

var execStartPathPattern = regexp.MustCompile(`path=(\S+)`)

type runner interface {
	Run(name string, args ...string) (string, error)
}

type SystemD struct {
	runner runner
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
		if v, verr := s.versionFromBinary(execStarts[name]); verr == nil {
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
//
// Units are matched by their Id= property rather than by output line
// position: a unit can have zero or several ExecStart= directives (each
// rendered on its own line), so the number of output lines does not equal
// the number of requested units and a positional match silently attributes
// one unit's binary to another.
func (s SystemD) execStarts(names []string) map[string]string {
	paths := make(map[string]string, len(names))
	if len(names) == 0 {
		return paths
	}

	args := append([]string{"show"}, names...)
	args = append(args, "--property=Id", "--property=ExecStart")

	out, err := s.runner.Run("systemctl", args...)
	if err != nil {
		return paths
	}

	for _, block := range strings.Split(out, "\n\n") {
		var id, path string
		for _, line := range strings.Split(block, "\n") {
			// systemctl orders properties by its own internal fixed order,
			// not by the order given on the command line, so Id= cannot be
			// assumed to come before ExecStart= within a block.
			if v, ok := strings.CutPrefix(line, "Id="); ok {
				id = v
				continue
			}
			if path == "" {
				if m := execStartPathPattern.FindStringSubmatch(line); len(m) == 2 {
					path = m[1] // first ExecStart= entry, for units with several
				}
			}
		}

		if id != "" {
			paths[id] = path
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
