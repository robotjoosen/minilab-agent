package discovery

import "github.com/robotjoosen/minilab-agent/pkg/domain"

type Aggregator struct {
	Systemd CommandRunner
	Docker  DockerClient
}

func (a *Aggregator) Discover() ([]domain.Service, error) {
	var services []domain.Service

	units, err := DiscoverSystemdUnits(a.Systemd)
	if err != nil {
		return nil, err
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

	containers, err := a.Docker.ListContainers()
	if err != nil {
		return nil, err
	}

	for _, c := range containers {
		services = append(services, domain.Service{
			Name:    c.Name,
			Type:    "docker",
			Up:      c.State == "running",
			Version: c.Image,
		})
	}

	return services, nil
}
