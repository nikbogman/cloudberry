# Homelab Control System

A control plane that lets the user wake, monitor, and suspend the main workload server (which hosts Docker-based services like Immich and other self-hosted apps) from anywhere on their Tailscale tailnet — without leaving the server running 24/7.

The full domain vocabulary — UI, Reachable, Wake, Suspend, Identity header, Deploy, Concern, Host group, etc. — is defined in [CONTEXT.md](CONTEXT.md). Read it before making non-trivial changes; this README stays intentionally high-level.

## How it fits together

Two physical devices on the same tailnet and the same LAN broadcast domain:

- **Pi Zero** — always on, low-power. Serves the UI and runs the Pi API.
- **Main server** — the workload machine (Immich, AI agents, etc.), spends most of its time suspended to RAM.

```
 Browser (tailnet) ──┬──> UI (Pi Zero, static)
                      │        │
                      │        ├──POST /wake────> Pi API (Pi Zero) ──WoL──> Main server
                      │        │
                      │        └──GET /health, POST /suspend──> Server API (main server)
                      │
                      └──any request──> Pi proxy (Pi Zero, Caddy) ──proxies + WoL──> Main server workloads
```

- **[UI](services/ui)** — the browser app. Polls reachability, offers Wake/Suspend buttons. The only user-facing surface.
- **[Pi API](services/pi-api)** — sends a deliberate Wake-on-LAN packet when the Wake button is pressed.
- **[Server API](services/server-api)** — exposes the Suspend action and the reachability health check the UI polls.
- **[Pi proxy](services/pi-proxy)** — a Caddy config on the Pi that reverse-proxies every workload service and transparently triggers the Pi API's wake action on any request while the server is asleep ([ADR-0012](docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md)).
- **[shared](services/shared)** — the auth, bind-safety, and Alloy-logging library both Flask apps depend on.

Auth for every control-plane endpoint is the `Tailscale-User-Login` header injected by `tailscale serve`: tailnet membership is the entire authorization boundary, with no separate allow-list ([ADR-0004](docs/adr/0004-tailnet-membership-authorization.md)). Both Flask apps are stateless — Grafana Alloy is where wake/suspend/reachability history actually lives.

Key architectural decisions are recorded as ADRs in [docs/adr/](docs/adr/), including why there's no server-side relay ([0001](docs/adr/0001-direct-browser-to-api-no-relay.md)), why suspend-to-RAM is the only sleep state ([0002](docs/adr/0002-suspend-only-no-shutdown.md)), why there are two wake triggers ([0005](docs/adr/0005-dual-wake-paths.md), revised by [0012](docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md)).

## Repo layout

```
CONTEXT.md              domain glossary — read this first
docs/adr/                architectural decisions
docs/agents/              how agent skills should use this repo's docs
services/
  ui/                      browser SPA (TypeScript + Vite)
  pi-api/                  Flask app on the Pi (Wake)
  server-api/              Flask app on the server (Suspend, health)
  shared/                  shared Python library for the two Flask apps
  pi-proxy/                Caddy config for transparent per-workload wake
pyinfra/                   declarative provisioning (pyinfra) for the Pi and server
.scratch/                  specs and issues for in-progress/planned features
```

## Status

The control system itself (UI, both the Pi API and Server API, the Pi proxy) is built per `.scratch/homelab-control-system/spec.md` — all five of its issues are implemented.

**Declarative provisioning (pyinfra) is built** per `.scratch/pyinfra-provisioning/spec.md` — all nine of its issues are implemented in [`pyinfra/`](pyinfra), which converges the Pi and the server to their declared state (Tailscale, Docker, Caddy, both the Pi API and Server API, the UI's static build) in one command. See [`pyinfra/README.md`](pyinfra/README.md) for usage, configuration, and the three-tier testing procedure. Tiers 2/3 of that procedure (disposable-container and real-device runs) and the Caddy binary's own provisioning still need running against real infrastructure — flagged explicitly in that README's Known gaps, not silently assumed done.

## Development

Each Python service (`pi-api`, `server-api`, `shared`) is an independent [uv](https://docs.astral.sh/uv/) project with its own virtualenv; the two apps pull in `shared` via an editable path dependency. `ui` is a separate npm project. There's no root-level build — work inside each service's directory. See each service's README for exact commands:

- [services/shared/README.md](services/shared/README.md)
- [services/pi-api/README.md](services/pi-api/README.md)
- [services/server-api/README.md](services/server-api/README.md)
- [services/ui/README.md](services/ui/README.md)
- [services/pi-proxy/README.md](services/pi-proxy/README.md)
- [pyinfra/README.md](pyinfra/README.md) — also an independent `uv` project, but not an installable package

## Working with this repo as an agent

`CLAUDE.md` documents three agent-skill conventions used throughout this repo: an issue tracker under `.scratch/<feature-slug>/` ([docs/agents/issue-tracker.md](docs/agents/issue-tracker.md)), a five-role triage-label vocabulary ([docs/agents/triage-labels.md](docs/agents/triage-labels.md)), and single-context domain docs — this file plus `docs/adr/` ([docs/agents/domain.md](docs/agents/domain.md)).
