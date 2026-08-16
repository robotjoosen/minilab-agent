// Package capabilities exposes the /capabilities endpoint, reporting which
// services (systemd units and docker containers) are discoverable on this
// host.
package capabilities

import (
	"encoding/json"
	"net/http"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/server"
)

// ServiceDiscoverer discovers the services running on this host.
type ServiceDiscoverer interface {
	Discover() ([]domain.Service, error)
}

type Handler struct {
	Discoverer ServiceDiscoverer
	Hostname   string
}

type response struct {
	Device   string           `json:"device"`
	Services []domain.Service `json:"services"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	services, err := h.Discoverer.Discover()
	if err != nil {
		server.ErrorResponse(w, r, http.StatusInternalServerError, "discovery failed", err.Error())
		return
	}

	body, err := json.Marshal(response{Device: h.Hostname, Services: services})
	if err != nil {
		server.ErrorResponse(w, r, http.StatusInternalServerError, "failed to marshal response", err.Error())
		return
	}

	server.SuccessResponse(w, string(body))
}
