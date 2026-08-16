# minilab-agent

![CI](https://github.com/robotjoosen/minilab-agent/actions/workflows/ci.yml/badge.svg)
![Go](https://img.shields.io/badge/go-1.26-00ADD8?logo=go&logoColor=white)
![platforms](https://img.shields.io/badge/platforms-linux%2Farm64%20%7C%20linux%2Farm-informational)

A small, read-only monitoring agent for a home lab of Raspberry Pis, Orange Pis, and whatever
else ends up on the shelf. One instance runs on each device, discovers what's actually running
there, and answers two HTTP endpoints so a central dashboard never has to guess.

Part of the Mini Lab Monitoring Dashboard initiative — this is Phase 1: pure observability, no
deploys, no config writes, nothing that can break your box.

## What it actually does

- **Discovers** local `systemd` units and Docker containers, on demand, no static registry to
  keep in sync
- **Listens** for the host's existing [`go-health-service`](https://github.com/robotjoosen/go-health-service)
  CPU/memory stats over AMQP — doesn't duplicate that work, just folds it in
- **Advertises itself** over mDNS so a dashboard can find it without a hand-maintained IP list
- **Serves** `GET /capabilities` (what's on this box) and `GET /metrics` (Prometheus text format)
- Stays up and answering even when Docker isn't installed, systemd discovery fails, or the
  message bus is unreachable — a monitoring agent that goes dark under partial failure is
  useless exactly when you need it most

```mermaid
flowchart LR
    subgraph device["this device — rocket / beanie / socks"]
        hs["go-health-service"]
        dk[("Docker")]
        sd["systemd"]
        agent["minilab-agent"]
    end
    dashboard["minilab-dashboard"]

    hs -- "health.ping · AMQP" --> agent
    dk -. "container list" .-> agent
    sd -. "unit list" .-> agent
    agent -- "mDNS advertise" --> dashboard
    dashboard -- "GET /capabilities" --> agent
    dashboard -- "GET /metrics" --> agent
```

## Endpoints

| Endpoint | Returns | Purpose |
|---|---|---|
| `GET /capabilities` | JSON | One-shot device interview: hostname + every discovered service |
| `GET /metrics` | Prometheus text | Host stats + per-service up/down + version, polled continuously |

<details>
<summary><code>GET /capabilities</code> — example response</summary>

```json
{
  "device": "rocket",
  "services": [
    { "Name": "nodered.service", "Type": "systemd", "Up": true, "Version": "2026-08-01T10:00:00Z" },
    { "Name": "ollama", "Type": "docker", "Up": true, "Version": "ollama/ollama:0.4.2" }
  ]
}
```

</details>

<details>
<summary><code>GET /metrics</code> — example response</summary>

```
minilab_host_cpu_percent{mode="user"} 12.4
minilab_host_cpu_percent{mode="system"} 3.1
minilab_host_cpu_percent{mode="idle"} 84.5
minilab_host_memory_bytes{state="used"} 1893000000
minilab_host_memory_bytes{state="free"} 500000000
minilab_host_memory_bytes{state="total"} 2393000000
minilab_service_up{name="nodered.service",type="systemd"} 1
minilab_service_up{name="ollama",type="docker"} 1
minilab_service_info{name="nodered.service",type="systemd",version="2026-08-01T10:00:00Z"} 1
minilab_service_info{name="ollama",type="docker",version="0.4.2"} 1
```

`version` lives on its own `service_info` metric rather than as a label on `service_up` — a
label whose value changes on every redeploy would mint a fresh Prometheus time series each time.

</details>

Errors on either endpoint come back as [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem
details, not a bare 500:

```json
{ "type": "about:blank", "title": "discovery failed", "status": 500, "detail": "...", "instance": "/metrics" }
```

## Quick start

On the target device:

```bash
curl -fsSL https://raw.githubusercontent.com/robotjoosen/minilab-agent/main/install.sh | bash
```

Or download and inspect it first (recommended for anything piping into a shell):

```bash
curl -fsSLO https://raw.githubusercontent.com/robotjoosen/minilab-agent/main/install.sh
chmod +x install.sh
./install.sh
```

It detects the device's architecture, pulls the matching binary from the
[latest release](https://github.com/robotjoosen/minilab-agent/releases), and walks through
setting it up as a systemd service — asking for the RabbitMQ URL and HTTP port, confirming
before every system-changing step (Docker group membership, writing the unit file,
enabling/starting the service).

To update later:

```bash
curl -fsSL https://raw.githubusercontent.com/robotjoosen/minilab-agent/main/update.sh | bash
```

`update.sh` preserves whatever's already configured and only asks for values genuinely missing
from the existing install (e.g. a setting a newer release added).

To remove it entirely:

```bash
curl -fsSL https://raw.githubusercontent.com/robotjoosen/minilab-agent/main/uninstall.sh | bash
```

`uninstall.sh` stops and disables the service, then confirms before removing the unit file, the
binary, and (optionally) the device's `docker` group membership it added.

None of `install.sh`, `update.sh`, or `uninstall.sh` are published as release assets — they're
fetched straight from `main` each time, same as the install command above. If you downloaded
`install.sh` to disk instead of piping it, fetch the other two the same way rather than expecting
them to already be sitting next to it.

## Configuration

Set as environment variables, or in a `.env` file next to the binary — see `.env.dist` for
defaults.

| Variable | Default | Description |
|---|---|---|
| `MODE` | `PROD` | `DEV` or `PROD` |
| `LOG_LEVEL` | `INFO` | slog level |
| `MESSAGE_BUS_URL` | `amqp://guest:guest@localhost:5672` | RabbitMQ connection |
| `MESSAGE_BUS_EXCHANGE` | `health` | Exchange `go-health-service` publishes to |
| `MESSAGE_BUS_ROUTINGKEY` | `health.ping` | Routing key to subscribe to |
| `HTTP_LISTEN_ADDR` | `:9100` | Where `/capabilities` and `/metrics` are served |
| `MDNS_SERVICE_NAME` | `_minilab-agent._tcp` | mDNS service type this agent advertises as |

## Development

Requires [Task](https://taskfile.dev) and Go (whatever version `go.mod` currently pins — it
floats as dependencies require newer minimums, deliberately not locked to match the rest of the
lab).

```bash
task build     # cross-compiles for linux/arm64 by default
go test ./... -race
gofmt -l .
```

### Project layout

```
cmd/app/                  entrypoint — env loading, wiring everything together
pkg/domain/               shared data structs only (HostStats, Service) — no logic
pkg/discovery/            CommandRunner/DockerClient interfaces, the discovery Aggregator, caching
pkg/handler/capabilities/ GET /capabilities
pkg/handler/metrics/      GET /metrics, Prometheus formatting
pkg/server/               generic HTTP server wrapper — Run/Stop, RFC 9457 error responses
pkg/healthstats/          AMQP subscriber + latest host-stats store
pkg/mdnsadvertise/        mDNS presence responder
pkg/env/                  env-to-struct loading (shared with go-health-service's own copy)
internal/docker/          Docker SDK client wrapper (transport detail, not part of the public API)
internal/exec/            os/exec wrapper for shelling out to systemctl (ditto)
```

`internal/` is reserved specifically for low-level transport wrappers like these two — everything
else lives under `pkg/`, matching the convention used across this lab's other services.

### Testing convention

Exported functions/methods are tested black-box, from `package foo_test` — importing the
package like any real caller would, so tests can't quietly depend on internal details of what's
supposed to be a clean API. Unexported helpers get their own white-box `package foo` test file,
since there's no other way to reach them. The two are never mixed in one file.

## Releasing

Push a tag matching `v*`:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The [release workflow](.github/workflows/release.yml) cross-compiles both architectures and
publishes them as `minilab-agent-linux-arm64` / `minilab-agent-linux-arm` — the exact names
`install.sh` and `update.sh` look for.

## Design notes

- **Partial discovery failures don't take down the whole endpoint.** If Docker isn't installed
  (two of the three named lab devices don't have it) or `systemctl` isn't available, that one
  source is skipped and logged — the other source's results still come back rather than a 500.
- **Discovery is cached** for a few seconds in front of the HTTP handlers, so a Prometheus scrape
  doesn't fork a `systemctl` process per unit on every single request.
- **The message bus connection never blocks startup.** If RabbitMQ is down, `/capabilities` and
  `/metrics` still answer immediately with whatever host stats are available (zero-valued until
  the first message arrives), rather than the whole agent hanging for minutes.

## Related

- [go-health-service](https://github.com/robotjoosen/go-health-service) — publishes the host
  stats this agent subscribes to
- minilab-dashboard — the (in-progress) central dashboard this agent exists to serve
