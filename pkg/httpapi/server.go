package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/robotjoosen/minilab-agent/pkg/domain"
)

type ServiceDiscoverer interface {
	Discover() ([]domain.Service, error)
}

type HostStatsProvider interface {
	Latest() domain.HostStats
}

type Server struct {
	Discoverer ServiceDiscoverer
	HostStats  HostStatsProvider
	Hostname   string
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/capabilities", s.handleCapabilities)
	mux.HandleFunc("/metrics", s.handleMetrics)
	return mux
}

type capabilitiesResponse struct {
	Device   string           `json:"device"`
	Services []domain.Service `json:"services"`
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	services, err := s.Discoverer.Discover()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(capabilitiesResponse{Device: s.Hostname, Services: services}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	services, err := s.Discoverer.Discover()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	io.WriteString(w, formatMetrics(s.HostStats.Latest(), services))
}
