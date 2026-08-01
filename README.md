# Homelab Control System

A control plane that lets the user wake, monitor, and suspend the main workload server (which hosts Docker-based services like Immich and other self-hosted apps) from anywhere on their Tailscale tailnet — without leaving the server running 24/7.

The full domain vocabulary — UI, Reachable, Wake, Suspend, Identity header, Deploy, Concern, Host group, etc. — is defined in [CONTEXT.md](CONTEXT.md). Read it before making non-trivial changes; this README stays intentionally high-level.

## How it fits together

Two physical devices on the same tailnet and the same LAN broadcast domain:

- **Gateway (Pi Zero)** — always on, low-power. Serves the UI and runs the Gateway API.
- **Compute host** — the workload machine (Immich, AI agents, etc.), spends most of its time suspended to RAM.

```
 Browser (tailnet) ──┬──> UI (Gateway, static, served by the Gateway API)
                      │
                      ├──POST /wake───────────────> Gateway API (Gateway) ──WoL──> Compute host
                      │
                      ├──any /server* request──────> Gateway API (Gateway) ──proxies + auto-WoL──> Compute host workloads
                      │
                      └──GET /health, POST /suspend──> Compute API (compute host)
```

- **[UI](control-plane/ui)** — the browser app. Polls reachability, offers Wake/Suspend buttons. The only user-facing surface.
- **[Gateway API](control-plane/README.md#gateway-api)** — one process on the Gateway that serves the UI's static files, sends a deliberate Wake-on-LAN packet when the Wake button is pressed, and reverse-proxies `/server*` to the Compute host, auto-waking it on a transport failure.
- **[Compute API](control-plane/README.md#compute-api)** — exposes the Suspend action and the reachability health check the UI polls.
- **[tailnet](control-plane/README.md#internal-tailnet)** and **[eventlog](control-plane/README.md#internal-eventlog)** — the identity-auth/bind-safety and Grafana Cloud event-logging libraries both Go binaries depend on.

Auth for every control-plane endpoint is the `Tailscale-User-Login` header injected by `tailscale serve`: tailnet membership is the entire authorization boundary, with no separate allow-list.

Key architectural decisions, data flow, and constraints are recorded in [`ARCHITECTURE.md`](ARCHITECTURE.md) — the canonical technical reference for this system.

## Repo layout

```
CONTEXT.md               domain glossary — read this first
ARCHITECTURE.md          canonical architecture/design reference
control-plane/
  ui/                       browser SPA (TypeScript + Vite)
  go.mod                    one Go module for the two control-plane binaries below
  cmd/gateway/              Go binary on the Gateway (serves the UI, Wake, /server* proxy + auto-wake)
  cmd/compute-api/          Go binary on the compute host (Suspend, health)
  internal/tailnet/         shared Go package: identity-header auth, bind-safety
  internal/eventlog/        shared Go package: Grafana Cloud event logging
provisioning/               declarative provisioning (pyinfra) for the gateway and compute host
deploy.sh                   forwards to provisioning/deploy.sh -- run from the repo root
.env.example                 checked-in template -- copy to .env (gitignored) and fill in real values
```

## Status

The control system (UI, both the Gateway API and Compute API) is fully implemented — see [`ARCHITECTURE.md`](ARCHITECTURE.md) for what it does and how it's put together.

**Declarative provisioning (pyinfra) is built** in [`provisioning/`](provisioning), which converges the gateway and the compute host to their declared state (Tailscale, Docker, both the Gateway API and Compute API, the UI's static build) in one command. See [`provisioning/README.md`](provisioning/README.md) for usage, configuration, and the three-tier testing procedure. Tiers 2/3 of that procedure (disposable-container and real-device runs) still need running against real infrastructure — flagged explicitly in that README's Known gaps, not silently assumed done.

## Development

The Gateway API and Compute API are one Go module (`control-plane/go.mod`) with the shared `tailnet`/`eventlog` packages and both binaries as packages underneath it — `go test ./...`/`go build ./...` from `control-plane/` covers both. `ui` is a separate npm project. `provisioning/` is an independent `uv` project (not an installable package). There's no single root-level build across all three. See:

- [control-plane/README.md](control-plane/README.md)
- [control-plane/ui/README.md](control-plane/ui/README.md)
- [provisioning/README.md](provisioning/README.md)

## Working with this repo as an agent

`CLAUDE.md` documents two agent-skill conventions used throughout this repo: an issue tracker under `docs/specs/<feature-slug>/` ([docs/agents/issue-tracker.md](docs/agents/issue-tracker.md)) and a five-role triage-label vocabulary ([docs/agents/triage-labels.md](docs/agents/triage-labels.md)).
