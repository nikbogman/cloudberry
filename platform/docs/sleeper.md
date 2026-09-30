# Sleeper

The machine that sleeps and runs the workloads, and the Sleeper API binary it
runs. Exposes a Reachable health check and the Suspend action. Fronted by its
own `tailscale serve` instance; only [Edge](edge.md) calls it, server-side, so
there's no CORS.

Vocabulary (Sleeper, Reachable, Suspend) is in [`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/sleeper/`](../internal/sleeper),
entrypoint [`cmd/sleeper-api/`](../cmd/sleeper-api).

## Routes

| Route | Behavior |
|---|---|
| `GET /health` | Requires the identity header. Returns `{"reachable": true}`. First time reachable, logs `reachability_changed` (see `reachabilityTracker` in [`handler.go`](../internal/sleeper/handler.go) — it can never observe going unreachable, since the process is asleep whenever that's true). |
| `POST /suspend` | Requires the identity header. Runs the configured suspend command (`systemctl suspend` by default, see [`suspend.go`](../internal/sleeper/suspend.go)), logs `suspend_requested`/`_succeeded`/`_failed`, returns `{"suspend": ...}`. |

Suspend-to-RAM only — the suspend command isn't configurable via env var, so
misconfiguration can't reintroduce a full shutdown.

### Suspend never waits for in-flight work

Userspace freezes mid-request: no FIN, no 503, just silence. Clients stall
until their own timeout; on resume the handler finishes the work and answers a
socket nobody is reading. A non-idempotent request can therefore run twice if
the client retries after the next Wake.

Both fixes are systemd's, so neither makes this binary learn about Docker.
Inhibitors are held per command, which suits one-shot jobs, not always-up
containers.

- **Seconds of grace** — run the job under `systemd-inhibit --mode=delay
  --what=sleep`, and raise `InhibitDelayMaxSec` (default 5s) in `logind.conf`.
  Suspend waits for the job, then proceeds anyway once the cap expires.
- **Refuse while busy** — run the job under `systemd-inhibit --what=sleep`
  (block mode) and add `--check-inhibitors=yes` to `DefaultSuspendCommand`;
  `systemctl suspend` defaults to `auto`, which honors block inhibitors only
  when invoked from a TTY. The UI already renders the resulting 500 as
  "Suspend failed". Pair it with `CombinedOutput()` in `suspend.go` so the log
  can tell busy from broken.

## Runtime environment

Read by the binary itself ([`internal/sleeper/config.go`](../internal/sleeper/config.go)),
which is the source of truth for these names and defaults. Bold variables are
required.

| Variable | Purpose |
|---|---|
| **`GRAFANA_CLOUD_LOKI_*`** | The same trio as the [Waker](waker.md#runtime-environment) — both binaries log to one endpoint. |
| `SLEEPER_API_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. |
| `SLEEPER_API_PORT` | Listen port (default `5000`). |

## Deployment

[`deploy/platform/sleeper_api.py`](../../deploy/platform/sleeper_api.py)
cross-compiles and ships only the binary, same as the Waker. `GOARCH` is read
from the device's real architecture (`common.DpkgArchitecture`), since `sleeper` isn't a
fixed known device.

It runs as a systemd unit rather than a container, so it stays controllable
while Docker itself redeploys. Docker is provisioned separately by
[`deploy/deploy_docker.py`](../../deploy/deploy_docker.py).

### Deploy-time variables

Set on the dev machine, in `platform/.env` (see `.env.example`).
`SLEEPER_API_HOST` is never set here — the binary's own loopback default
applies. Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| `SLEEPER_API_PORT` | `5000` | Listen port; pinned explicitly for the same `tailscale serve` reason as the Waker's port |
| **`GRAFANA_CLOUD_LOKI_*`** | — | The same trio as the [Waker](waker.md#deploy-time-variables) — both binaries log to one endpoint |

This file also runs `tailscale serve` for
`$SLEEPER_API_PORT`, same mechanism and version caveat as the Waker.

See [`deploy.md`](../../docs/deploy.md) for how to run a Deploy and how
it's tested.
