// Package metrics exposes the /metrics endpoint in Prometheus text
// exposition format, reporting host resource usage and discovered service
// status.
package metrics

import (
	"io"
	"net/http"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
	"github.com/robotjoosen/minilab-agent/pkg/server"
)

// ServiceDiscoverer discovers the services running on this host.
type ServiceDiscoverer interface {
	Discover() ([]domain.Service, error)
}

// HostStatsProvider reports the most recently observed host resource usage.
type HostStatsProvider interface {
	Latest() domain.HostStats
}

type Handler struct {
	Discoverer ServiceDiscoverer
	HostStats  HostStatsProvider
}

// Handle serves Prometheus text exposition format. It does not use
// server.SuccessResponse, since that hardcodes a JSON content type.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	services, err := h.Discoverer.Discover()
	if err != nil {
		server.ErrorResponse(w, r, http.StatusInternalServerError, "discovery failed", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	io.WriteString(w, formatMetrics(h.HostStats.Latest(), services))
}
