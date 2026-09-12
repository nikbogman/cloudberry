# Compute

The machine that sleeps and runs the workloads, and the Compute API binary it
runs. Exposes a Reachable health check and the Suspend action. Fronted by its
own `tailscale serve` instance — a distinct origin from the [UI](ui.md), hence the CORS
allow-list.

Vocabulary (Compute, Reachable, Suspend) is in [`CONTEXT.md`](../CONTEXT.md).

Code: [`internal/compute/`](../internal/compute),
entrypoint [`cmd/compute-api/`](../cmd/compute-api).

## Routes

| Route | Behavior |
|---|---|
| `GET /health` | Requires the identity header. Returns `{"reachable": true}`. First time reachable, logs `reachability_changed` (see `reachabilityTracker` in [`handler.go`](../internal/compute/handler.go) — it can never observe going unreachable, since the process is asleep whenever that's true). |
| `POST /suspend` | Requires the identity header. Runs the configured suspend command (`systemctl suspend` by default, see [`suspend.go`](../internal/compute/suspend.go)), logs `suspend_requested`/`_succeeded`/`_failed`, returns `{"suspend": ...}`. |

Suspend-to-RAM only — the suspend command isn't configurable via env var, so
misconfiguration can't reintroduce a full shutdown.

Workload containers are not this binary's concern: each publishes its own port
and is reached directly, so Compute API never talks to the Docker daemon.

## Runtime environment

Read by the binary itself ([`internal/compute/config.go`](../internal/compute/config.go)),
which is the source of truth for these names and defaults. Bold variables are
required.

| Variable | Purpose |
|---|---|
| **`UI_ORIGIN`** | Origin allowed by the CORS policy. |
| **`GRAFANA_CLOUD_LOKI_*`** | The same trio as the [Waker](waker.md#runtime-environment) — both binaries log to one endpoint. |
| `COMPUTE_API_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. |
| `COMPUTE_API_PORT` | Listen port (default `5000`). |

## Deployment

[`deploy/deploy_compute_api.py`](../deploy/deploy_compute_api.py) cross-compiles
and ships only the binary, same as the Waker. `GOARCH` is read from the
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
| `COMPUTE_API_PORT` | `5000` | Listen port; pinned explicitly for the same `tailscale serve` reason as the Waker's port |
| **`GRAFANA_CLOUD_LOKI_*`** | — | The same trio as the [Waker](waker.md#deploy-time-variables) — both binaries log to one endpoint |

`UI_ORIGIN` (the CORS allow-list entry) is derived from `InventorySettings`, not
a separate secret. This file also runs `tailscale serve` for
`$COMPUTE_API_PORT`, same mechanism and version caveat as the Waker.

See [`deploy.md`](deploy.md) for how to run a Deploy and how it's tested.
