# waker

Runs on raspberry, the always-on Pi Zero W. Its one job: send Wake-on-LAN to
blackberry. [edge](../edge/README.md) forwards the UI's Wake to it over the tailnet; it
stays on the Pi because the magic packet is a LAN broadcast, which needs
raspberry and blackberry on one L2 broadcast domain.

Vocabulary (raspberry, Wake, Identity header) is in [`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/waker/`](../../internal/waker),
entrypoint [`cmd/waker/`](.).

## Routes

| Route | Behavior |
|---|---|
| `POST /wake` | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `BLACKBERRY_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |

## Environment

Read by [`internal/waker/config.go`](../../internal/waker/config.go), the source
of truth for names and defaults. Set in the repo root `.env` on the dev
machine; `task waker` passes them to the unit. Bold variables are required.

| Variable | Purpose |
|---|---|
| **`BLACKBERRY_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `WAKER_PORT` | Listen port (default `5000`). The deploy pins it, since `tailscale serve` must point at it. |
| `WAKER_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. The deploy never sets it. |

## Deployment

`task waker` cross-compiles for the Pi Zero W, ships the binary and a systemd
unit (the device never runs a Go toolchain), and runs `tailscale serve` for
`$WAKER_PORT`. See [`DEPLOY.md`](../../DEPLOY.md).
