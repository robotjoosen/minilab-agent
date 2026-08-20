package discovery

import (
	"errors"
	"log/slog"

	"github.com/robotjoosen/minilab-agent/internal/docker"
	"github.com/robotjoosen/minilab-agent/internal/systemd"
	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

// Discover gathers services from Docker and systemd independently and is
// tolerant of a partial failure: if one leg fails, it is logged and
// discovery continues with whatever the other leg produced. An error is
// only returned when both legs fail, since at that point there is nothing
// useful to report.
func Discover() (domain.Services, error) {
	var s = make(domain.Services, 0)

	dl, dErr := listDockerContainers()
	if dErr != nil {
		slog.Error("docker discovery failed", slog.String("error", dErr.Error()))
	}

	sl, sErr := listSystemdServices()
	if sErr != nil {
		slog.Error("systemd discovery failed", slog.String("error", sErr.Error()))
	}

	if dErr != nil && sErr != nil {
		return nil, errors.Join(dErr, sErr)
	}

	s = append(s, dl...)
	s = append(s, sl...)

	return s, nil
}

func listDockerContainers() (domain.Services, error) {
	d, err := docker.New()
	if err != nil {
		return nil, err
	}

	l, err := d.ListContainers()
	if err != nil {
		return nil, err
	}

	s := make(domain.Services, 0, len(l))
	for _, c := range l {
		s = append(s, domain.ServiceItem{
			Name:    c.Name,
			Type:    "docker",
			State:   new(domain.Status).Parse(c.State),
			Version: c.Image,
		})
	}

	return s, nil
}

func listSystemdServices() (domain.Services, error) {
	d := systemd.New()

	l, err := d.GetProcesses()
	if err != nil {
		return nil, err
	}

	s := make(domain.Services, 0, len(l))
	for _, c := range l {
		s = append(s, domain.ServiceItem{
			Name:    c.Name,
			Type:    "systemd",
			State:   new(domain.Status).Parse(c.State),
			Version: c.Version,
		})
	}

	return s, nil
}
