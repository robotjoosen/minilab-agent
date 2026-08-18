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

	var services Services
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		var state = StateUnknown
		if fields[2] == StateActive.String() {
			state = StateActive
		}

		name := fields[0]
		execStart := ""
		if showOut, showErr := s.runner.Run("systemctl", "show", name, "--property=ExecStart", "--value"); showErr == nil {
			if m := execStartPathPattern.FindStringSubmatch(showOut); len(m) == 2 {
				execStart = m[1]
			}
		}

		var version string
		if v, verr := s.versionFromBinary(execStart); verr == nil {
			version = v
		}

		services = append(services, Service{
			Name:    name,
			Type:    "systemd",
			State:   state.String(),
			Version: version,
		})
	}

	return services, nil
}

func (s SystemD) versionFromBinary(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	return info.ModTime().UTC().Format(time.RFC3339), nil
}
