# Architecture

Primary architecture reference for the homelab control plane: current architecture, key decisions, and constraints. Domain vocabulary: [`CONTEXT.md`](CONTEXT.md). Documents the system as it exists today, not superseded designs.

## System Overview

**Purpose**: wake, monitor, and suspend the main workload machine ("Compute") from anywhere on the tailnet, so it can host Docker services (Immich, AI agents, etc.) without running 24/7.

Two physical devices on the same tailnet and LAN broadcast domain.

<p align="center"><img src="homelab.drawio.png" alt="High-level architecture"></p>

| Component | Responsibility |
|---|---|
| UI | Browser SPA. Polls Compute reachability, offers Wake/Suspend. |
| Gateway API | Serves the UI's static build, sends WoL on `POST /wake`, reverse-proxies `/server/*` to Compute with auto-wake-on-failure. |
| Compute API | Exposes `GET /health` and `POST /suspend` on the Compute host. |
| Provisioning (pyinfra) | Converges both devices to their target state — Tailscale, Docker, both binaries. |

## Architecture

### Major components

```
control-plane/
  ui/                    TypeScript + Vite SPA
  cmd/gateway/            Gateway API entrypoint (env → wiring → ListenAndServe)
  cmd/compute-api/        Compute API entrypoint
  internal/gateway/       Gateway API handlers: static serving, /wake, /server/* proxy
  internal/compute/       Compute API handlers: CORS, /health, /suspend, suspend command
  internal/tailnet/       shared: identity-header auth, bind-safety
  internal/eventlog/      shared: Grafana Cloud (Loki) event logger
  internal/httpresponse/  shared: JSON response + response-copy helpers
  internal/env/           shared: env-var lookup helpers
provisioning/             pyinfra: inventory, settings, one deploy_<x>.py per concern
```

### How they interact

- The browser talks to **two origins directly** — Gateway (UI + Gateway API, same-origin) and Compute API (separate origin) — no server-side relay. Compute API's only extra cost is a one-entry CORS allow-list.
- `/server/*` reverse-proxies through the Gateway API to a single fixed upstream on Compute. Which path reaches which Docker service is entirely that upstream's concern.
- Gateway API's auto-wake is an **in-process call**: the proxy's error handler and the `/wake` handler both call the same private `doWake` directly.
- Provisioning depends on control-plane and ui; neither depends on provisioning.

## Domain Model

See [`CONTEXT.md`](CONTEXT.md).

### Key entities (code level)

- `gateway.Handler` / `compute.Handler` — one per binary, own the route table and dependencies.
- `reachabilityTracker` (`internal/compute`) — logs `reachability_changed` once per process lifetime via `sync.Once`; can't observe going *un*reachable, since the process is asleep whenever that's true.
- `autoWakeThrottler` (`internal/gateway`) — single `lastRun` timestamp, gating repeat wakes to once per 90s.
- `GrafanaCloudLogger` (`internal/eventlog`) — sole implementation of both apps' `EventLogger` interface.

### Key workflows

1. **Reachability polling** — UI polls `GET /health`; renders Reachable/Unreachable, toggles Wake/Suspend.
2. **Deliberate wake** — Wake button → `POST /wake` → WoL packet + `wake_*` events.
3. **Automatic wake** — `/server/*` request while asleep → transport failure → throttled wake → retried every 1s for up to 60s → first success written through.
4. **Suspend** — Suspend button → `POST /suspend` → suspend command + `suspend_*` events.
5. **Deploy** — `./deploy.sh` → pyinfra over SSH → builds/ships binaries + UI, converges systemd and `tailscale serve` on both devices.

## Data Flow

### Request paths

```mermaid
sequenceDiagram
    participant B as Browser
    participant G as Gateway API
    participant C as Compute API/host

    B->>G: GET / (poll asset load)
    G-->>B: UI static build (go:embed)

    loop every 12s
        B->>C: GET /health (Identity header)
        C-->>B: 200 {reachable: true}
    end

    B->>G: POST /wake (Identity header or loopback)
    G->>C: WoL magic packet (UDP broadcast)
    G->>G: event log
    G-->>B: 200/500 {wake: ...}

    B->>G: /server/* (any method)
    alt Compute reachable
        G->>C: proxy request (path-stripped)
        C-->>G: real response (incl. 502s) passed through untouched
        G-->>B: response
    else transport failure
        G->>G: throttled auto-wake trigger
        loop retry every 1s, up to 60s
            G->>C: retry request
        end
        G-->>B: first success, or 502 after 60s
    end

    B->>C: POST /suspend (Identity header)
    C->>C: run suspend command + event log
    C-->>B: 200/500 {suspend: ...}
```

### External integrations

| Integration | Purpose | Notes |
|---|---|---|
| **Tailscale** (`tailscale serve`) | TLS termination + identity injection; WoL is a raw UDP broadcast, not over Tailscale. | Entire authn/authz boundary is tailnet membership. |
| **Grafana Cloud Loki push API** | Structured event log (wake/suspend/reachability). | No local collector. Failures are logged and swallowed. |

## Key Design Decisions

- **No server-side relay between the UI's two backend origins** — avoids the Compute API trusting a forwarded identity claim over the real header.
- **Suspend-to-RAM only, no full shutdown** — WoL after ACPI S5 is unreliable across BIOS/NIC configs; the suspend command isn't configurable via env var.
- **WoL requires the same L2 broadcast domain** — accepted rather than building a cross-subnet relay.
- **Tailnet membership is the entire authorization boundary** — no separate allow-list.
- **Compute owns all workload routing** — the Gateway proxies one blind route, so new workload services never touch this repo.
- **Events ship directly to Grafana Cloud, no local collector** — not worth a self-hosted Alloy instance for low-volume audit events.
- **Both apps are Go, built and shipped as binaries** — no interpreter/dependency tree on-device; suits low-resource hardware like the Pi Zero.
- **The UI is also built off-device** — the Pi Zero is single-core/low-memory; only `dist/` ships.
- **systemd, not Docker, for both apps** — the Compute API must stay controllable while Docker itself redeploys.
- **Deploys are manually triggered only** — no CI/cron/hook, to avoid unattended-deploy risk on a two-device setup.
- **Secrets are plain dev-machine environment variables** — an encrypted-secrets workflow is unneeded for a single-operator setup.

## Extension Points

- **New control-plane action**: add a route in the relevant `NewHandler`, define a small interface for external effects, fake it in tests. Follow `handleWake`/`handleSuspend`'s shape: log `_requested`, perform the effect, log `_succeeded`/`_failed`, write JSON.
- **New workload service behind `/server/*`**: no change here — routing is Compute's downstream proxy's responsibility.
- **New deployed component**: add `provisioning/deploy_<name>.py`, gate with `common.has_device_role(...)`, add settings if needed, `local.include(...)` it from `deploy.py`.
- **New event type**: call `logger.SendEvent(eventType, outcome, identity, extra)` — no schema migration; labels are fixed, everything else rides in the log line.

## Constraints

**Assumptions**
- Gateway and Compute stay on the same L2 broadcast domain; no cross-subnet WoL path or error signal if that stops being true.
- Single-operator, two-device scale — not multi-tenant, no fleet of compute hosts, no concurrent deploys.
- Exactly one compute upstream: the auto-wake throttle and `/server/*` proxy both assume a single fixed target.

**Performance**
- UI polls `/health` every 12s with a 5s client timeout, so a powered-off host doesn't strand the UI in "Checking…".
- A cold `/server/*` request holds the connection up to 60s (1s retries) waiting for Compute to wake.
- Automatic wake is throttled to once per 90s regardless of request volume.

**Security**
- Auth is tailnet membership via `Tailscale-User-Login` — no app-level user store or allow-list.
- One exception: `/wake` also accepts a `127.0.0.1` caller with no identity header, scoped to loopback, for the in-process auto-wake trigger.
- `tailnet.AssertTailnetOnlyBind` is a structural backstop: both binaries refuse to start if their bind address isn't loopback or tailnet.
- Compute API's CORS policy allows exactly one origin.

**Known limitations**
- No error path if Gateway and Compute split onto different broadcast domains/VLANs — WoL packets just stop arriving.
- `reachabilityTracker` can only observe Compute *becoming* reachable, never unreachable, since the process is asleep whenever that transition happens.
- The Grafana Cloud pipeline has no retry/buffering — a transient outage drops the event. Accepted for low-stakes audit events.
- Bind-address enforcement is a runtime check per binary; pyinfra doesn't verify it at provisioning time.
- The downstream workload proxy `/server/*` forwards to doesn't exist in this repo — a manual prerequisite, out of scope.
