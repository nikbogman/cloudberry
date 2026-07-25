# Homelab Control System

A control plane that lets the user wake, monitor, and suspend the main workload server (which hosts Docker-based services like Immich and other self-hosted apps) from anywhere on their Tailscale tailnet — without leaving the server running 24/7.

The full domain vocabulary — UI, Reachable, Wake, Suspend, Identity header, Deploy, Concern, Host group, etc. — is defined in [CONTEXT.md](CONTEXT.md). Read it before making non-trivial changes; this README stays intentionally high-level.

## How it fits together

Two physical devices on the same tailnet and the same LAN broadcast domain:

- **Gateway (Pi Zero)** — always on, low-power. Serves the UI and runs the Gateway API.
- **Compute host** — the workload machine (Immich, AI agents, etc.), spends most of its time suspended to RAM.

```
 Browser (tailnet) ──┬──> UI (Gateway, static)
                      │        │
                      │        ├──POST /wake────> Gateway API (Gateway) ──WoL──> Compute host
                      │        │
                      │        └──GET /health, POST /suspend──> Compute API (compute host)
                      │
                      └──any request──> Gateway proxy (Gateway, Caddy) ──proxies + WoL──> Compute host workloads
```

- **[UI](services/ui)** — the browser app. Polls reachability, offers Wake/Suspend buttons. The only user-facing surface.
- **[Gateway API](services/README.md#gateway-api)** — sends a deliberate Wake-on-LAN packet when the Wake button is pressed.
- **[Compute API](services/README.md#compute-api)** — exposes the Suspend action and the reachability health check the UI polls.
- **[Gateway proxy](services/gateway-proxy)** — a Caddy config on the Gateway that reverse-proxies every workload service and transparently triggers the Gateway API's wake action on any request while the compute host is asleep ([ADR-0012](docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md)).
- **[tailnet](services/README.md#internal-tailnet)** and **[eventlog](services/README.md#internal-eventlog)** — the identity-auth/bind-safety and Grafana Cloud event-logging libraries both Go binaries depend on.

Auth for every control-plane endpoint is the `Tailscale-User-Login` header injected by `tailscale serve`: tailnet membership is the entire authorization boundary, with no separate allow-list ([ADR-0004](docs/adr/0004-tailnet-membership-authorization.md)).

Key architectural decisions are recorded as ADRs in [docs/adr/](docs/adr/), including why there's no server-side relay ([0001](docs/adr/0001-direct-browser-to-api-no-relay.md)), why suspend-to-RAM is the only sleep state ([0002](docs/adr/0002-suspend-only-no-shutdown.md)), why there are two wake triggers ([0005](docs/adr/0005-dual-wake-paths.md), revised by [0012](docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md)).

## Repo layout

```
CONTEXT.md              domain glossary — read this first
docs/adr/                architectural decisions
docs/agents/              how agent skills should use this repo's docs
services/
  ui/                      browser SPA (TypeScript + Vite)
  go.mod                   one Go module for the two control-plane binaries below
  cmd/gateway-api/         Go binary on the Gateway (Wake)
  cmd/compute-api/         Go binary on the compute host (Suspend, health)
  internal/tailnet/        shared Go package: identity-header auth, bind-safety
  internal/eventlog/       shared Go package: Grafana Cloud event logging
  gateway-proxy/           Caddy config for transparent per-workload wake
pyinfra/                   declarative provisioning (pyinfra) for the gateway and compute host
.scratch/                  specs and issues for in-progress/planned features
```

## Status

The control system itself (UI, both the Gateway API and Compute API, the Gateway proxy) is built per `.scratch/homelab-control-system/spec.md` — all five of its issues are implemented.

**Declarative provisioning (pyinfra) is built** per `.scratch/pyinfra-provisioning/spec.md` — all nine of its issues are implemented in [`pyinfra/`](pyinfra), which converges the gateway and the compute host to their declared state (Tailscale, Docker, Caddy, both the Gateway API and Compute API, the UI's static build) in one command. See [`pyinfra/README.md`](pyinfra/README.md) for usage, configuration, and the three-tier testing procedure. Tiers 2/3 of that procedure (disposable-container and real-device runs) and the Caddy binary's own provisioning still need running against real infrastructure — flagged explicitly in that README's Known gaps, not silently assumed done.

## Development

The Gateway API and Compute API are one Go module (`services/go.mod`) with the shared `tailnet`/`eventlog` packages and both binaries as packages underneath it — `go test ./...`/`go build ./...` from `services/` covers both. `ui` is a separate npm project. `pyinfra/` is an independent `uv` project (not an installable package). There's no single root-level build across all three. See:

- [services/README.md](services/README.md)
- [services/ui/README.md](services/ui/README.md)
- [pyinfra/README.md](pyinfra/README.md)

## Working with this repo as an agent

`CLAUDE.md` documents three agent-skill conventions used throughout this repo: an issue tracker under `.scratch/<feature-slug>/` ([docs/agents/issue-tracker.md](docs/agents/issue-tracker.md)), a five-role triage-label vocabulary ([docs/agents/triage-labels.md](docs/agents/triage-labels.md)), and single-context domain docs — this file plus `docs/adr/` ([docs/agents/domain.md](docs/agents/domain.md)).
