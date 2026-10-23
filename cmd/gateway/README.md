# gateway

Runs on raspberry, the always-on Pi Zero W. It sends Wake-on-LAN to
blackberry in two cases:

- **The UI's Wake.** [edge](../edge/README.md) forwards it to `POST /wake` over the tailnet.
- **A request to a stack route flagged `wake`** in [`routes.json`](routes.json).
  gateway holds the request until the route's upstream accepts a TCP connection
  on the LAN, then reverse-proxies it. After `idle_minutes` with no route traffic it calls
  hostd `/suspend`, but only if it auto-woke blackberry.

It lives on raspberry because the magic packet is a LAN broadcast, which needs
raspberry and blackberry on one L2 broadcast domain. Rationale:
[DESIGN.md](../../DESIGN.md#design-decisions). Vocabulary is in
[`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/gateway/`](../../internal/gateway), entrypoint [`cmd/gateway/`](.).

## Routes

| Route | Behavior |
|---|---|
| `POST /wake` on `$GATEWAY_PORT` (published on 443) | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `BLACKBERRY_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |
| Each stack route, on its own `port` | Reverse-proxies to `upstream`, waking blackberry first if the route has `wake`. Logs `autowake_*` and `autosuspend_*`. |

## Routes file

`routes.json` is shipped to `/etc/gateway/routes.json` by `task gateway`.

| Field | Meaning |
|---|---|
| `idle_minutes` | Quiet time before gateway suspends blackberry. |
| `name` | Label for logs. |
| `port` | Loopback listen port and the HTTPS port published on raspberry's hostname. Unique per route. Never 443: `/wake` owns it. |
| `upstream` | Absolute URL of the stack, reachable from raspberry. |
| `public` | `true` publishes with `tailscale funnel` (port must be 8443 or 10000). `false` publishes with `tailscale serve`, and gateway requires the identity header. |
| `wake` | `true` sends WoL and holds the request while blackberry is asleep. `false` returns 502 while it's asleep. |

## Environment

Read by [`internal/gateway/config.go`](../../internal/gateway/config.go), the
source of truth for names and defaults. Set in the repo root `.env` on the dev
machine; `task gateway` passes them to the unit. Bold variables are required.

| Variable | Purpose |
|---|---|
| **`BLACKBERRY_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`HOSTD_URL`** | hostd's tailnet URL, as raspberry reaches it. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `GATEWAY_PORT` | `/wake` listen port (default `5000`). The deploy pins it, since `tailscale serve` must point at it. |
| `GATEWAY_HOST` | `/wake` bind address (default `127.0.0.1`). Must be loopback or tailnet. The deploy never sets it. |
| `GATEWAY_ROUTES` | Routes file path (default `/etc/gateway/routes.json`). The deploy never sets it. |

## Deployment

`task gateway` ships the routes file, cross-compiles for the Pi Zero W, ships
the binary and a systemd unit (the device never runs a Go toolchain), and runs
`tailscale serve` for `$GATEWAY_PORT` on 443. gateway itself runs
`tailscale serve`/`funnel` for each stack route at startup. Funnel must be
enabled for raspberry's node in the tailnet policy. See
[`DEPLOY.md`](../../DEPLOY.md).

Replacing the old waker and gate units on raspberry, once:
`task cleanup DEVICE=RASPBERRY OLD=waker`, `task cleanup DEVICE=RASPBERRY OLD=gate`,
then `task gateway`.
