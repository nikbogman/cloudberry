# Homelab Control System

Wakes, monitors, and suspends the main workload server (Docker services like Immich) from anywhere on the tailnet, without leaving it running 24/7.

Domain vocabulary (UI, Reachable, Wake, Suspend, Identity header, Deploy, Host group, etc.) is defined in [CONTEXT.md](CONTEXT.md).

## How it fits together

Two devices on the same tailnet and LAN broadcast domain:

- **Gateway (Pi Zero)** — always on, low-power. Serves the UI and runs the Gateway API.
- **Compute host** — runs the workloads, suspended to RAM most of the time.

The pieces running on them:

- **[UI](control-plane/ui)** — browser app. Polls reachability, offers Wake/Suspend.
- **[Gateway API](control-plane/README.md#gateway-api)** — serves the UI, sends Wake-on-LAN, reverse-proxies `/server*` to Compute with auto-wake.
- **[Compute API](control-plane/README.md#compute-api)** — exposes Suspend, Hold, and the health check the UI polls; also Compute's own reverse proxy to workload containers, behind an idle watcher that auto-suspends after an hour of no use.
- **[tailnet](control-plane/README.md#internaltailnet)** / **[eventlog](control-plane/README.md#internaleventlog)** — shared identity-auth/bind-safety and Grafana Cloud logging packages.

Auth is the `Tailscale-User-Login` header injected by `tailscale serve` — tailnet membership is the entire authorization boundary.

See [`docs/architecture.md`](docs/architecture.md) for design decisions, data flow, and constraints.

## Repo layout

```
CONTEXT.md               domain glossary
docs/architecture.md     architecture/design reference
control-plane/
  ui/                       browser SPA (TypeScript + Vite)
  go.mod                    one Go module for both binaries below
  cmd/gateway/              Gateway API (UI, Wake, /server* proxy)
  cmd/compute-api/          Compute API (Suspend, Hold, health, container proxy, idle watcher)
  internal/tailnet/         shared: identity-header auth, bind-safety
  internal/eventlog/        shared: Grafana Cloud event logging
deploy/                  declarative provisioning (pyinfra)
deploy.sh                forwards to deploy/deploy.sh
.env.example              template -- copy to .env (gitignored)
```

## Status

Fully implemented; Deploy ([`deploy/`](deploy)) converges both devices in one command. Outstanding gaps: [`deploy/README.md`](deploy/README.md#known-gaps) and [`docs/architecture.md`](docs/architecture.md#constraints).

## Development

One Go module covers both binaries (`go build ./...`/`go test ./...` from `control-plane/`). `ui/` is a separate npm project; `deploy/` an independent `uv` project. No root-level build.

- [control-plane/README.md](control-plane/README.md)
- [control-plane/ui/README.md](control-plane/ui/README.md)
- [deploy/README.md](deploy/README.md)
