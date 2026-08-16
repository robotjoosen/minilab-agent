package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robotjoosen/minilab-agent/pkg/server"
)

func TestSuccessResponse(t *testing.T) {
	rec := httptest.NewRecorder()

	server.SuccessResponse(rec, `{"ok":true}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}

	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestErrorResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/some/path", nil)
	rec := httptest.NewRecorder()

	server.ErrorResponse(rec, req, http.StatusBadRequest, "bad request", "something was wrong")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json content type, got %q", ct)
	}

	var problem server.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	want := server.Problem{
		Type:     "about:blank",
		Title:    "bad request",
		Status:   http.StatusBadRequest,
		Detail:   "something was wrong",
		Instance: "/some/path",
	}
	if problem != want {
		t.Fatalf("unexpected problem body: %+v, want %+v", problem, want)
	}
}

func TestNotFoundResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()

	s := &server.Server{}
	s.NotFoundResponse(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	var problem server.Problem
	if err := json.NewDecoder(rec.Body).Decode(&problem); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if problem.Status != http.StatusNotFound || problem.Instance != "/nope" {
		t.Fatalf("unexpected problem body: %+v", problem)
	}
}

func TestInitialiseRoutesUnmatchedPathUsesNotFoundResponse(t *testing.T) {
	s := &server.Server{}
	mux := s.InitialiseRoutes(map[string]http.HandlerFunc{
		"GET /known": func(w http.ResponseWriter, r *http.Request) {
			server.SuccessResponse(w, `{"known":true}`)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unmatched route, got %d", rec.Code)
	}
}

func TestRunAndStopLifecycle(t *testing.T) {
	const port = 19173

	s := &server.Server{Port: port}
	s.InitialiseRoutes(map[string]http.HandlerFunc{
		"GET /ping": func(w http.ResponseWriter, r *http.Request) {
			server.SuccessResponse(w, `{"pong":true}`)
		},
	})

	s.Run()

	var resp *http.Response
	var err error
	for i := 0; i < 50; i++ {
		resp, err = http.Get("http://127.0.0.1:19173/ping")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to reach server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if string(body) != `{"pong":true}` {
		t.Fatalf("unexpected body: %s", body)
	}

	s.Stop()

	if _, err := http.Get("http://127.0.0.1:19173/ping"); err == nil {
		t.Fatal("expected server to be unreachable after Stop()")
	}
}
