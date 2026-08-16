package discovery

import (
	"os"
	"regexp"
	"strings"
	"time"
)

type CommandRunner interface {
	Run(name string, args ...string) (stdout string, err error)
}

type Unit struct {
	Name      string
	ExecStart string
	Active    bool
}

var execStartPathPattern = regexp.MustCompile(`path=(\S+)`)

func DiscoverSystemdUnits(runner CommandRunner) ([]Unit, error) {
	out, err := runner.Run("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--plain")
	if err != nil {
		return nil, err
	}

	var units []Unit
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		name := fields[0]
		active := fields[2] == "active"

		execStart := ""
		if showOut, showErr := runner.Run("systemctl", "show", name, "--property=ExecStart", "--value"); showErr == nil {
			if m := execStartPathPattern.FindStringSubmatch(showOut); len(m) == 2 {
				execStart = m[1]
			}
		}

		units = append(units, Unit{Name: name, ExecStart: execStart, Active: active})
	}

	return units, nil
}

func VersionFromBinary(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	return info.ModTime().UTC().Format(time.RFC3339), nil
}
