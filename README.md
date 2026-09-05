# Homelab Control System

Wakes, monitors, and suspends the main workload server (Docker services like
Immich) from anywhere on the tailnet, without leaving it running 24/7.

Two devices on the same tailnet and LAN broadcast domain:

- **Gateway (Pi Zero W)** — always on, low-power. Serves the UI, sends
  Wake-on-LAN, reverse-proxies `/server*` to Compute with auto-wake.
- **Compute** — runs the workloads, suspended to RAM most of the time.
  Exposes Suspend, Hold, and the health check the UI polls, plus its own
  reverse proxy to workload containers behind an idle watcher.

Auth is the `Tailscale-User-Login` header injected by `tailscale serve` —
tailnet membership is the entire authorization boundary.

## Docs

| Document | Owns |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Design decisions, data flow, constraints, shared Go packages |
| [CONTEXT.md](CONTEXT.md) | Domain glossary — the vocabulary everything else uses |
| [docs/gateway.md](docs/gateway.md) | Gateway API: routes, environment, deployment |
| [docs/compute.md](docs/compute.md) | Compute API: routes, idle watcher, environment, deployment |
| [docs/ui.md](docs/ui.md) | Browser SPA: behavior, build-time config |
| [docs/deploy.md](docs/deploy.md) | pyinfra: inventory, running a Deploy, the three test tiers |

## Repo layout

```
ARCHITECTURE.md          design reference
CONTEXT.md               domain glossary
docs/                    per-service docs; agents/ is agent-tooling contract
control-plane/
  go.mod                    one Go module for both binaries
  cmd/gateway/              Gateway API entrypoint
  cmd/compute-api/          Compute API entrypoint
  internal/gateway/         Gateway API logic
  internal/compute/         Compute API logic
  internal/tailnet/         shared: identity-header auth, bind-safety
  internal/eventlog/        shared: Grafana Cloud event logging
  internal/httpresponse/    shared: JSON response helpers
  internal/env/             shared: env-var lookup helpers
  ui/                       browser SPA (TypeScript + Vite)
deploy/                  declarative provisioning (pyinfra)
deploy.sh                forwards to deploy/deploy.sh
.env.example             template -- copy to .env (gitignored)
```

No root-level build: three independent toolchains.

## Development

```sh
# Go, both binaries
cd control-plane && go vet ./... && go test ./...

# UI
cd control-plane/ui && npm install
npm run dev          # Vite dev server
npm run test         # vitest
npm run typecheck    # tsc --noEmit
npm run build        # tsc && vite build -> dist/

# Deploy tooling
cd deploy && uv sync
```

Run a binary locally with its environment, e.g.:

```sh
COMPUTE_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
COMPUTE_HOST=main-server.tailnet \
COMPUTE_PROXY_PORT=8081 \
go run ./cmd/gateway
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
