# Compute

The machine that sleeps and runs the workloads, and the Compute API binary it
runs. Exposes a Reachable health check, the Suspend action, Hold, and its own
reverse proxy to workload Docker containers. Fronted by its own `tailscale
serve` instance — a distinct origin from the [UI](ui.md), hence the CORS
allow-list.

Vocabulary (Compute, Reachable, Suspend, Hold) is in [`CONTEXT.md`](../CONTEXT.md).

Code: [`control-plane/internal/compute/`](../control-plane/internal/compute),
entrypoint [`control-plane/cmd/compute-api/`](../control-plane/cmd/compute-api).

## Routes

| Route | Behavior |
|---|---|
| `GET /health` | Requires the identity header. Returns `{"reachable": true}`. First time reachable, logs `reachability_changed` (see `reachabilityTracker` in [`handler.go`](../control-plane/internal/compute/handler.go) — it can never observe going unreachable, since the process is asleep whenever that's true). |
| `POST /suspend` | Requires the identity header. Runs the configured suspend command (`systemctl suspend` by default, see [`suspend.go`](../control-plane/internal/compute/suspend.go)), logs `suspend_requested`/`_succeeded`/`_failed`, returns `{"suspend": ...}`. |
| `POST /hold` | Requires the identity header or a `127.0.0.1` caller. Acquires or renews a fixed 30-minute hold that blocks automatic suspend, logs `hold_acquired`, returns `{"hold": "acquired"}` (see [`activitysignals.go`](../control-plane/internal/compute/activitysignals.go)). |
| `DELETE /hold` | Requires the identity header or a `127.0.0.1` caller. Releases an active hold, logs `hold_released`; a no-op (no event) if nothing was held, so a script can call it unconditionally on exit. |
| `GET /status` | Requires the identity header (no loopback exception — read diagnostic, not an automation target). Returns idle duration, hold state + remaining TTL, and whether dry-run mode is on. |
| `/{prefix}/*` (everything else) | Not identity-gated, mirroring the Gateway's `/server/` route. Reverse-proxies to whichever Docker container carries a `homelab.route={prefix}` label, path-stripped (see [`proxy.go`](../control-plane/internal/compute/proxy.go)). A container is only routable if it publishes a port to the host. A real backend response (even an error status) passes through untouched; an unmatched path 404s; a container-discovery failure 502s. Each successful proxy is timestamped, readable via `Handler.LastProxiedAt()` — an activity signal for the idle watcher. |

Suspend-to-RAM only — the suspend command isn't configurable via env var, so
misconfiguration can't reintroduce a full shutdown.

Container discovery talks to the local Docker daemon via the official Docker Go
SDK ([`docker.go`](../control-plane/internal/compute/docker.go)), injected as the
`ContainerRuntime` interface — same construction pattern as
`Suspender`/`EventLogger`, faked in tests with no real daemon required.

## Idle watcher

[`idlewatcher.go`](../control-plane/internal/compute/idlewatcher.go) runs
alongside the handler, polling once a minute. Compute counts as active if
container CPU is above baseline, a request was proxied within the current
window, or a Hold is active — container running-state alone is never the signal,
since workload containers stay up regardless of use.

After `COMPUTE_API_IDLE_TIMEOUT` with none of those true, it calls the same
`Suspender.Suspend()` that `POST /suspend` uses, logging
`suspend_auto_triggered`/`_succeeded`/`_failed` instead of the manual event
types. With `COMPUTE_API_AUTOSUSPEND_DRY_RUN` set, it runs the identical
decision logic but logs `suspend_auto_dry_run` and keeps cycling instead of
actually suspending — for validating the watcher live without disrupting the
machine.

## Runtime environment

Read by the binary itself ([`internal/compute/config.go`](../control-plane/internal/compute/config.go)),
which is the source of truth for these names and defaults. Bold variables are
required.

| Variable | Purpose |
|---|---|
| **`UI_ORIGIN`** | Origin allowed by the CORS policy. |
| **`GRAFANA_CLOUD_LOKI_*`** | The same trio as the [Gateway](gateway.md#runtime-environment) — both binaries log to one endpoint. |
| `COMPUTE_API_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. |
| `COMPUTE_API_PORT` | Listen port (default `5000`). |
| `COMPUTE_API_IDLE_TIMEOUT` | Idle time before auto-suspend triggers (default `1h`). |
| `COMPUTE_API_CPU_BASELINE_PERCENT` | Container CPU% above which counts as activity (default `2`). |
| `COMPUTE_API_AUTOSUSPEND_DRY_RUN` | If `true`, the idle watcher logs instead of actually suspending (default `false`). |

## Deployment

[`deploy/deploy_compute_api.py`](../deploy/deploy_compute_api.py) cross-compiles
and ships only the binary, same as the Gateway. `GOARCH` is read from the
device's real architecture (`common.DpkgArchitecture`), since `compute` isn't a
fixed known device.

It runs as a systemd unit rather than a container, so it stays controllable
while Docker itself redeploys. Docker is provisioned separately by
[`deploy/deploy_docker.py`](../deploy/deploy_docker.py).

### Deploy-time variables

Set on the dev machine. `COMPUTE_API_HOST` is never set here — the binary's own
loopback default applies. Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| `COMPUTE_API_PORT` | `5000` | Listen port; pinned explicitly for the same `tailscale serve` reason as the Gateway's port |
| **`GRAFANA_CLOUD_LOKI_*`** | — | The same trio as the [Gateway](gateway.md#deploy-time-variables) — both binaries log to one endpoint |

`UI_ORIGIN` (the CORS allow-list entry) is derived from `InventorySettings`, not
a separate secret. This file also runs `tailscale serve` for
`$COMPUTE_API_PORT`, same mechanism and version caveat as the Gateway.

See [`deploy.md`](deploy.md) for how to run a Deploy and how it's tested.
