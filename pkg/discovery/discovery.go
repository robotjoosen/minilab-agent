package discovery

import (
	"errors"

	"github.com/robotjoosen/minilab-agent/internal/docker"
	"github.com/robotjoosen/minilab-agent/internal/systemd"
	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

func Discover() (domain.Services, error) {
	var err error
	var s = make(domain.Services, 0)

	dl, dErr := listDockerContainers()
	if dErr != nil {
		err = errors.Join(err, dErr)
	}

	sl, sErr := listSystemdServices()
	if sErr != nil {
		err = errors.Join(err, sErr)
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

	l, _ := d.ListContainers()
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
