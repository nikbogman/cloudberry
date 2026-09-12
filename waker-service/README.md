# Waker Service

Wakes, monitors, and suspends the Sleeper — the box running the actual
workloads (Docker services like Immich) — from anywhere on the tailnet,
without leaving it running 24/7.

Two devices on the same tailnet and LAN broadcast domain:

- **Waker (Pi Zero W)** — always on, low-power. Serves the UI and sends
  Wake-on-LAN. Not in the workload traffic path: it's on Wi-Fi, so routing
  workload traffic through it isn't worth the hop.
- **Sleeper** — runs the workloads, suspended to RAM most of the time.
  Exposes Suspend and the health check the UI polls. Workload containers
  publish their own ports; browsers reach them directly on the tailnet.

Auth is the `Tailscale-User-Login` header injected by `tailscale serve` —
tailnet membership is the entire authorization boundary.

## Docs

| Document | Owns |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Design decisions, data flow, constraints, shared Go packages |
| [docs/waker.md](docs/waker.md) | Waker API: routes, environment, deployment |
| [docs/sleeper.md](docs/sleeper.md) | Sleeper API: routes, environment, deployment |
| [docs/ui.md](docs/ui.md) | Browser SPA: behavior, runtime config |
| [../CONTEXT.md](../CONTEXT.md) | Domain glossary — the vocabulary everything else uses |
| [../docs/deploy.md](../docs/deploy.md) | pyinfra: inventory, running a Deploy, the three test tiers |

## Layout

```
ARCHITECTURE.md          design reference
docs/                    one file per surface (waker, sleeper, ui)
go.mod                   one Go module for both binaries
cmd/waker-api/           Waker API entrypoint
cmd/sleeper-api/         Sleeper API entrypoint
internal/waker/          Waker API logic
internal/sleeper/        Sleeper API logic
internal/tailnet/        shared: identity-header auth, bind-safety
internal/eventlog/       shared: Grafana Cloud event logging
internal/httpresponse/   shared: JSON response helpers
internal/env/            shared: env-var lookup helpers
ui/                      browser SPA (static HTML/JS/CSS, no build step);
                         ui.go is just its go:embed declaration
dev-ui.sh                fast-iteration UI dev server (see docs/ui.md)
.env.example             template -- copy to .env (gitignored)
```

One Go module, no build step for the UI, no Node. Provisioning lives at
the repo root in [../deploy/](../deploy/) — shared across every homelab
service — and builds these binaries from here via `module_dir`.

## Development

```sh
# Go, both binaries (from this folder)
go vet ./... && go test ./...

# UI -- static files served from disk; edits show on refresh
./dev-ui.sh
```

Run a binary locally with its environment, e.g.:

```sh
SLEEPER_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
SLEEPER_API_URL=https://main-server.tailnet \
go run ./cmd/waker-api
```

## Deploying

From the repo root:

```sh
./deploy.sh waker --dry       # preview just the Waker
./deploy.sh sleeper_api       # converge just the Sleeper API
./deploy.sh                   # converge every device
```

This service's Deploy files are grouped in
[`../deploy/waker-service/`](../deploy/waker-service/); the inventory, shared
helpers, and homelab-wide infrastructure stay at the top of `../deploy/`.

Secrets split in two: this service's own vars in `waker-service/.env`
(see `.env.example`), the device addresses in the repo root `.env`.
`deploy/deploy.sh` sources both. See [../docs/deploy.md](../docs/deploy.md).

## Status

Fully implemented; one command converges both devices. Outstanding gaps:
[../docs/deploy.md](../docs/deploy.md#known-gaps) and
[ARCHITECTURE.md](ARCHITECTURE.md#constraints).
