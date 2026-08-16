package discovery

import (
	"errors"
	"log/slog"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

type Aggregator struct {
	Systemd CommandRunner
	Docker  DockerClient
}

// Discover gathers services from systemd and Docker independently and is
// tolerant of a partial failure: if one leg fails, it is logged and discovery
// continues with whatever the other leg produced. An error is only returned
// when both legs fail, since at that point there is nothing useful to report.
func (a *Aggregator) Discover() ([]domain.Service, error) {
	var services []domain.Service

	units, systemdErr := DiscoverSystemdUnits(a.Systemd)
	if systemdErr != nil {
		slog.Error("systemd discovery failed", slog.String("error", systemdErr.Error()))
	}

	for _, u := range units {
		version := ""
		if u.ExecStart != "" {
			if v, verr := VersionFromBinary(u.ExecStart); verr == nil {
				version = v
			}
		}

		services = append(services, domain.Service{
			Name:    u.Name,
			Type:    "systemd",
			Up:      u.Active,
			Version: version,
		})
	}

	containers, dockerErr := a.Docker.ListContainers()
	if dockerErr != nil {
		slog.Error("docker discovery failed", slog.String("error", dockerErr.Error()))
	}

	for _, c := range containers {
		services = append(services, domain.Service{
			Name:    c.Name,
			Type:    "docker",
			Up:      c.State == "running",
			Version: c.Image,
		})
	}

	if systemdErr != nil && dockerErr != nil {
		return nil, errors.Join(systemdErr, dockerErr)
	}

	return services, nil
}
