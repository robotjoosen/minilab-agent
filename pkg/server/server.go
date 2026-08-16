// Package server is a http router wrapper for basic server functionality.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	Port   int
	mux    *http.ServeMux
	server *http.Server
}

// Problem is an RFC 9457 ("Problem Details for HTTP APIs") response body.
// RFC 9457 obsoletes RFC 7807.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

func (s *Server) InitialiseRoutes(routeHandlers map[string]http.HandlerFunc) *http.ServeMux {
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/", s.NotFoundResponse)

	for pattern, routeFunc := range routeHandlers {
		s.mux.HandleFunc(pattern, routeFunc)
	}

	return s.mux
}

func (s *Server) Run() {
	slog.Info("starting server",
		slog.Int("port", s.Port),
	)

	s.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.Port),
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server stopped", slog.String("error", err.Error()))
		}
	}()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return
	}
}

func (s *Server) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	slog.Warn("no response available",
		slog.Int("status_code", http.StatusNotFound),
		slog.String("path", r.RequestURI),
	)

	ErrorResponse(w, r, http.StatusNotFound, "not found", "no handler defined for path")
}

func SuccessResponse(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w, content)
}

// ErrorResponse writes an RFC 9457 "problem+json" response. status is the
// actual HTTP status code being returned, and r is used to populate the
// Problem's Instance field with the request path.
func ErrorResponse(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	msg, err := json.Marshal(Problem{
		Type:     "about:blank",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
	})
	if err != nil {
		slog.Error(err.Error())

		return
	}

	fmt.Fprint(w, string(msg))
}
