# HTTP server layer retrofit — report

Migrated `pkg/httpapi` to a generic `pkg/server` wrapper plus two small handler
packages under `pkg/handler/`, following the `api-message-generator` pattern,
with the RFC 9457 deviation and `internal/`-avoidance rules from the task spec.

## Files built, file-by-file

### `pkg/server/server.go` (new)
Generic HTTP server wrapper, copied in structure from
`api-message-generator/pkg/server/server.go` with the required deviations:

- `Server{Port int; mux *http.ServeMux; server *http.Server}`
- `InitialiseRoutes(map[string]http.HandlerFunc) *http.ServeMux` — registers a
  catch-all `"/"` -> `NotFoundResponse`, then each supplied pattern.
- `Run()` — starts `http.Server{Addr: fmt.Sprintf(":%d", s.Port), ReadHeaderTimeout: 5s}`
  listening in a background goroutine.
- `Stop()` — graceful shutdown via `context.WithTimeout(..., 5*time.Second)`.
- `NotFoundResponse` — default 404 handler, now routed through the new
  `ErrorResponse`.
- `SuccessResponse(w, content)` — unchanged from the reference: sets
  `Content-Type: application/json`, writes 200 + the given string.
- **`Problem` struct (replaces the reference's misnamed `RFC7808`)**:
  ```go
  type Problem struct {
      Type     string `json:"type"`
      Title    string `json:"title"`
      Status   int    `json:"status"`
      Detail   string `json:"detail"`
      Instance string `json:"instance,omitempty"`
  }
  ```
- **`ErrorResponse(w http.ResponseWriter, r *http.Request, status int, title, detail string)`**
  — writes `Content-Type: application/problem+json`, sets the response status
  to the given `status` (not hardcoded 500 like the reference), and populates
  `Instance` from `r.URL.Path`.

### `pkg/server/server_test.go` (new)
- `TestSuccessResponse` — asserts 200, `application/json`, and body content.
- `TestErrorResponse` — asserts status code, `application/problem+json`
  content type, and the full decoded `Problem` shape (`type`, `title`,
  `status`, `detail`, `instance`) against an exact expected value.
- `TestNotFoundResponse` — asserts 404 and that `Instance` reflects the
  request path.
- `TestInitialiseRoutesUnmatchedPathUsesNotFoundResponse` — routes an
  unmatched path through the real mux and confirms it falls through to 404.
- `TestRunAndStopLifecycle` — real `Run()`/`Stop()` smoke test against a fixed
  ephemeral-range port (19173), polling until the server accepts connections,
  hitting a real registered route over the wire, then calling `Stop()` twice
  and confirming the port becomes unreachable afterward. Matches the rigor of
  `pkg/mdnsadvertise/advertise_test.go`'s real-lifecycle smoke test.

### `pkg/handler/capabilities/handler.go` (new)
- Defines `ServiceDiscoverer` interface locally (`Discover() ([]domain.Service, error)`),
  matching the existing local-interface convention in `pkg/discovery/docker.go`
  and `pkg/discovery/systemd.go`.
- `Handler{Discoverer ServiceDiscoverer; Hostname string}`
- `Handle(w, r)` — calls `Discoverer.Discover()`; on error, calls
  `server.ErrorResponse(w, r, http.StatusInternalServerError, "discovery failed", err.Error())`;
  on success, marshals `{device, services}` JSON and calls
  `server.SuccessResponse(w, string(body))`.

### `pkg/handler/capabilities/handler_test.go` (new)
- `TestHandleReturnsCapabilities` — migrated from the old
  `TestCapabilitiesEndpoint`, now calling `Handler.Handle` directly instead of
  going through a mux/`Routes()`.
- `TestHandleDiscoveryErrorProducesRFC9457Problem` — migrated/extended from
  the old `TestCapabilitiesEndpointDiscoveryError`; now asserts the full
  RFC 9457 `Problem` body (`type`, `title`, `status`, `detail`, `instance`)
  rather than just the status code.

### `pkg/handler/metrics/handler.go` (new)
- Defines its own local `ServiceDiscoverer` and `HostStatsProvider`
  interfaces (duplicated rather than shared, per spec).
- `Handler{Discoverer ServiceDiscoverer; HostStats HostStatsProvider}`
- `Handle(w, r)` — on discovery error, `server.ErrorResponse(...)`; on
  success, sets `Content-Type: text/plain; version=0.0.4` itself and writes
  the formatted text via `io.WriteString` — **does not** call
  `server.SuccessResponse` (which would force `application/json`).

### `pkg/handler/metrics/format.go` (new)
`formatMetrics(host domain.HostStats, services []domain.Service) string`
moved verbatim from `pkg/httpapi/metrics.go`, kept unexported, as a sibling
file in the same package as `handler.go`.

### `pkg/handler/metrics/handler_test.go` (new)
Internal test file (`package metrics`, not `metrics_test`) so it can exercise
the unexported `formatMetrics` directly alongside handler-level tests:
- `TestHandleReturnsPrometheusMetrics` — migrated from `TestMetricsEndpoint`;
  additionally asserts the `text/plain; version=0.0.4` content type.
- `TestHandleDiscoveryErrorProducesRFC9457Problem` — new; asserts the full
  RFC 9457 `Problem` body on a discovery failure.
- `TestFormatMetrics` / `TestFormatMetricsSortsServicesByName` /
  `indexOf` — migrated verbatim from `pkg/httpapi/metrics_test.go`.

### `cmd/app/main.go` (modified)
- Import swap: removed `pkg/httpapi`, `net/http` retained (needed for
  `http.HandlerFunc`), added `pkg/handler/capabilities`, `pkg/handler/metrics`,
  `pkg/server`.
- Replaced `server := &httpapi.Server{Discoverer, HostStats, Hostname}` with:
  - `discoverer := discovery.NewCachingDiscoverer(...)` (unchanged logic,
    now a standalone local var instead of an embedded field)
  - `capsHandler := &capabilities.Handler{Discoverer: discoverer, Hostname: hostname}`
  - `metricsHandler := &metrics.Handler{Discoverer: discoverer, HostStats: store}`
- `portNum` derivation from `e.HTTPListenAddr` unchanged.
- Replaced the raw `*http.Server` + manual goroutine + manual
  `httpServer.Shutdown(shutdownCtx)` with:
  ```go
  srv := &server.Server{Port: portNum}
  srv.InitialiseRoutes(map[string]http.HandlerFunc{
      "GET /capabilities": capsHandler.Handle,
      "GET /metrics":      metricsHandler.Handle,
  })
  srv.Run()
  ...
  <-ctx.Done()
  srv.Stop()
  ```
- **Naming collision avoided**: the old local variable named `server`
  (holding `*httpapi.Server`) is gone; the new `*server.Server` instance is
  named `srv`, so it does not shadow the imported `server` package.

### `pkg/httpapi/*` — deleted
Deleted `server.go`, `metrics.go`, `server_test.go`, `metrics_test.go` and the
now-empty `pkg/httpapi/` directory. Confirmed via
`grep -rn "pkg/httpapi" --include="*.go" .` (run from repo root) that no file
in the repo references the old package anymore (grep exit code 1, i.e. zero
matches).

`pkg/domain`, `pkg/discovery`, `pkg/healthstats`, `pkg/mdnsadvertise`,
`pkg/env` were not touched.

## Verification

### `go build ./...`
Exit 0, no output.

### `go vet ./...`
Exit 0, no output.

### `gofmt -l .`
Exit 0, no output (no files need formatting).

### `go test ./... -v -race`
```
?   	github.com/robotjoosen/minilab-agent/cmd/app	[no test files]
?   	github.com/robotjoosen/minilab-agent/internal/docker	[no test files]
?   	github.com/robotjoosen/minilab-agent/internal/exec	[no test files]
=== RUN   TestAggregatorDiscoverCombinesSystemdAndDocker
--- PASS: TestAggregatorDiscoverCombinesSystemdAndDocker (0.00s)
=== RUN   TestAggregatorDiscoverToleratesDockerFailure
--- PASS: TestAggregatorDiscoverToleratesDockerFailure (0.00s)
=== RUN   TestAggregatorDiscoverToleratesSystemdFailure
--- PASS: TestAggregatorDiscoverToleratesSystemdFailure (0.00s)
=== RUN   TestAggregatorDiscoverReturnsErrorWhenBothFail
--- PASS: TestAggregatorDiscoverReturnsErrorWhenBothFail (0.00s)
=== RUN   TestCachingDiscovererServesFromCacheWithinTTL
--- PASS: TestCachingDiscovererServesFromCacheWithinTTL (0.00s)
=== RUN   TestCachingDiscovererRefreshesAfterTTLExpires
--- PASS: TestCachingDiscovererRefreshesAfterTTLExpires (0.00s)
=== RUN   TestDockerClientInterfaceIsSatisfiedByFake
--- PASS: TestDockerClientInterfaceIsSatisfiedByFake (0.00s)
=== RUN   TestDiscoverSystemdUnits
--- PASS: TestDiscoverSystemdUnits (0.00s)
=== RUN   TestVersionFromBinaryUsesModTime
--- PASS: TestVersionFromBinaryUsesModTime (0.00s)
PASS
ok  	github.com/robotjoosen/minilab-agent/pkg/discovery	1.373s
?   	github.com/robotjoosen/minilab-agent/pkg/domain	[no test files]
?   	github.com/robotjoosen/minilab-agent/pkg/env	[no test files]
=== RUN   TestHandleReturnsCapabilities
--- PASS: TestHandleReturnsCapabilities (0.00s)
=== RUN   TestHandleDiscoveryErrorProducesRFC9457Problem
--- PASS: TestHandleDiscoveryErrorProducesRFC9457Problem (0.00s)
PASS
ok  	github.com/robotjoosen/minilab-agent/pkg/handler/capabilities	1.592s
=== RUN   TestHandleReturnsPrometheusMetrics
--- PASS: TestHandleReturnsPrometheusMetrics (0.00s)
=== RUN   TestHandleDiscoveryErrorProducesRFC9457Problem
--- PASS: TestHandleDiscoveryErrorProducesRFC9457Problem (0.00s)
=== RUN   TestFormatMetrics
--- PASS: TestFormatMetrics (0.00s)
=== RUN   TestFormatMetricsSortsServicesByName
--- PASS: TestFormatMetricsSortsServicesByName (0.00s)
PASS
ok  	github.com/robotjoosen/minilab-agent/pkg/handler/metrics	2.544s
=== RUN   TestParseHealthMessage
--- PASS: TestParseHealthMessage (0.00s)
=== RUN   TestParseHealthMessageInvalidJSON
--- PASS: TestParseHealthMessageInvalidJSON (0.00s)
=== RUN   TestStoreUpdateAndLatest
--- PASS: TestStoreUpdateAndLatest (0.00s)
PASS
ok  	github.com/robotjoosen/minilab-agent/pkg/healthstats	2.309s
=== RUN   TestStartAndCloseDoesNotError
--- PASS: TestStartAndCloseDoesNotError (0.01s)
PASS
ok  	github.com/robotjoosen/minilab-agent/pkg/mdnsadvertise	2.101s
=== RUN   TestSuccessResponse
--- PASS: TestSuccessResponse (0.00s)
=== RUN   TestErrorResponse
--- PASS: TestErrorResponse (0.00s)
=== RUN   TestNotFoundResponse
--- PASS: TestNotFoundResponse (0.00s)
=== RUN   TestInitialiseRoutesUnmatchedPathUsesNotFoundResponse
--- PASS: TestInitialiseRoutesUnmatchedPathUsesNotFoundResponse (0.00s)
=== RUN   TestRunAndStopLifecycle
--- PASS: TestRunAndStopLifecycle (0.02s)
PASS
ok  	github.com/robotjoosen/minilab-agent/pkg/server	1.870s
```
All packages pass, race detector enabled, exit code 0.

### Cross-compile: `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ./...`
Exit 0, no output.

### Cross-compile: `GOOS=linux GOARCH=arm CGO_ENABLED=0 go build ./...`
Exit 0, no output.

## Self-review against the task's required deviations

1. **RFC 9457, not RFC7808** — `pkg/server.Problem` has exactly the
   specified fields/JSON tags (`type`, `title`, `status`, `detail`,
   `instance,omitempty`). `Status` is a caller-supplied parameter, not
   hardcoded. `Instance` is populated from `r.URL.Path`. `ErrorResponse`
   signature is `(w http.ResponseWriter, r *http.Request, status int, title, detail string)`
   exactly as specified. Confirmed via `TestErrorResponse` (exact struct
   equality against the expected `Problem` value) in both `pkg/server` and
   both handler packages.
2. **Everything new under `pkg/`, nothing under `internal/`** — confirmed;
   `internal/docker` and `internal/exec` are untouched, and the new code
   lives in `pkg/server/`, `pkg/handler/capabilities/`, `pkg/handler/metrics/`.
3. **`/metrics` bypasses `SuccessResponse`** — `metrics.Handler.Handle` sets
   `Content-Type: text/plain; version=0.0.4` directly and writes via
   `io.WriteString`; `capabilities.Handler.Handle` uses
   `server.SuccessResponse`. Confirmed by the content-type assertions in both
   handler tests.
4. **Method-scoped routes** — `cmd/app/main.go` registers
   `"GET /capabilities"` and `"GET /metrics"` (not bare paths).

## Confirmation pkg/httpapi is gone

`pkg/httpapi/` directory removed (all 4 files). `grep -rn "pkg/httpapi" --include="*.go" .`
from the repo root returns zero matches (exit code 1).

## Notes / judgment calls

- Kept `formatMetrics` as an unexported function in a sibling `format.go`
  file within `pkg/handler/metrics` (task explicitly left this as "your
  call").
- Put the RFC 9457 problem-shape assertions in *all three* of
  `pkg/server`, `pkg/handler/capabilities`, and `pkg/handler/metrics` tests,
  rather than picking just one location, since the task said "your call
  which is cleaner, but cover it somewhere" — extra coverage seemed strictly
  better here given how central the error shape is to the whole retrofit.
- The `pkg/server` `Run()`/`Stop()` lifecycle test uses a fixed port (19173)
  in the ephemeral-ish range rather than `:0`, since the reference/parent
  design (`fmt.Sprintf(":%d", s.Port)`) doesn't expose the OS-assigned port
  back to the caller. This mirrors `pkg/mdnsadvertise`'s test, which also
  hardcodes a port (9100).
