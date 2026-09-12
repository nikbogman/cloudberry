# Homelab Control System

Wakes, monitors, and suspends the main workload server (Docker services like
Immich) from anywhere on the tailnet, without leaving it running 24/7.

Two devices on the same tailnet and LAN broadcast domain:

- **Gateway (Pi Zero W)** — always on, low-power. Serves the UI and sends
  Wake-on-LAN. Not in the workload traffic path: it's on Wi-Fi, so routing
  workload traffic through it isn't worth the hop.
- **Compute** — runs the workloads, suspended to RAM most of the time.
  Exposes Suspend and the health check the UI polls, plus its own reverse
  proxy to workload containers. Browsers reach it directly on the tailnet.

Auth is the `Tailscale-User-Login` header injected by `tailscale serve` —
tailnet membership is the entire authorization boundary.

## Docs

| Document | Owns |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Design decisions, data flow, constraints, shared Go packages |
| [CONTEXT.md](CONTEXT.md) | Domain glossary — the vocabulary everything else uses |
| [docs/gateway.md](docs/gateway.md) | Gateway API: routes, environment, deployment |
| [docs/compute.md](docs/compute.md) | Compute API: routes, environment, deployment |
| [docs/ui.md](docs/ui.md) | Browser SPA: behavior, runtime config |
| [docs/deploy.md](docs/deploy.md) | pyinfra: inventory, running a Deploy, the three test tiers |

## Repo layout

```
ARCHITECTURE.md          design reference
CONTEXT.md               domain glossary
docs/                    per-service docs; agents/ is agent-tooling contract
go.mod                   one Go module for both binaries
cmd/gateway-api/         Gateway API entrypoint
cmd/compute-api/         Compute API entrypoint
internal/gateway/        Gateway API logic
internal/compute/        Compute API logic
internal/tailnet/        shared: identity-header auth, bind-safety
internal/eventlog/       shared: Grafana Cloud event logging
internal/httpresponse/   shared: JSON response helpers
internal/env/            shared: env-var lookup helpers
ui/                      browser SPA (static HTML/JS/CSS, no build step);
                         ui.go is just its go:embed declaration
deploy/                  declarative provisioning (pyinfra)
deploy.sh                forwards to deploy/deploy.sh
dev-ui.sh                fast-iteration UI dev server (see docs/ui.md)
.env.example             template -- copy to .env (gitignored)
```

Two toolchains: Go at the root, uv in `deploy/`. The UI is static files —
no build step, no Node.

## Development

```sh
# Go, both binaries (repo root)
go vet ./... && go test ./...

# UI -- static files served from disk; edits show on refresh
./dev-ui.sh

# Deploy tooling
cd deploy && uv sync
```

Run a binary locally with its environment, e.g.:

```sh
COMPUTE_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
COMPUTE_API_URL=https://main-server.tailnet \
go run ./cmd/gateway-api
```

## Deploying

```sh
./deploy.sh --dry    # preview
./deploy.sh          # converge both devices
```

See [docs/deploy.md](docs/deploy.md).

## Status

Fully implemented; one command converges both devices. Outstanding gaps:
[docs/deploy.md](docs/deploy.md#known-gaps) and
[ARCHITECTURE.md](ARCHITECTURE.md#constraints).
