# hostd

blackberry's own daemon. Exposes a Reachable health check and the Suspend
action. Fronted by its own `tailscale serve` instance; only [edge](../edge/README.md)
calls it.

Vocabulary (blackberry, Reachable, Suspend) is in [`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/hostd/`](../../internal/hostd),
entrypoint [`cmd/hostd/`](.).

## Routes

| Route | Behavior |
|---|---|
| `GET /health` | Requires the identity header. Returns `{"reachable": true}`. Logs `reachability_changed` the first time; it can never observe going unreachable, since the process is asleep whenever that's true. |
| `POST /suspend` | Requires the identity header. Runs `systemctl suspend` (see [`suspend.go`](../../internal/hostd/suspend.go)), logs `suspend_requested`/`_succeeded`/`_failed`, returns `{"suspend": ...}`. |

The suspend command isn't configurable via env var, so misconfiguration can't
reintroduce a full shutdown.

### Suspend never waits for in-flight work

Userspace freezes mid-request: no FIN, no 503, just silence. Clients stall
until their own timeout; on resume the handler finishes the work and answers a
socket nobody is reading. A non-idempotent request can therefore run twice if
the client retries after the next Wake. Possible fixes:
[`.scratch/suspend-inflight-work.md`](../../.scratch/suspend-inflight-work.md).

## Environment

Read by [`internal/hostd/config.go`](../../internal/hostd/config.go), the source
of truth for names and defaults. Set in the repo root `.env` on the dev
machine; `task hostd` passes them to the unit. Bold variables are required.

| Variable | Purpose |
|---|---|
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `HOSTD_PORT` | Listen port (default `5000`). The deploy pins it, since `tailscale serve` must point at it. |
| `HOSTD_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. The deploy never sets it. |

## Deployment

`task hostd` builds for blackberry's architecture (read over ssh), ships the
binary and a systemd unit, and runs `tailscale serve` for `$HOSTD_PORT`. It's
a systemd unit rather than a container so it stays controllable while Docker
itself redeploys. See [`DEPLOY.md`](../../DEPLOY.md).
