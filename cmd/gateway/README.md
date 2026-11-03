# gateway

Runs on raspberry, the always-on Pi Zero W. It sends Wake-on-LAN to
blackberry in two cases:

- **The UI's Wake.** The browser calls `POST /wake` directly, over the tailnet.
- **A request to a stack route flagged `wake`** in the [routes file](#routes-file).
  gateway holds the request until the route's upstream accepts a TCP connection
  on the LAN, then reverse-proxies it. Once every route has been quiet for its
  `idle_minutes` it calls hostd `/suspend`, but only if it auto-woke blackberry.

Both are switchable at runtime from the UI (`/settings`). The switches live in
memory: a gateway restart turns both back on.

It lives on raspberry because the magic packet is a LAN broadcast, which needs
raspberry and blackberry on one L2 broadcast domain. Rationale:
[DESIGN.md](../../DESIGN.md#design-decisions). Vocabulary is in
[`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/gateway/`](../../internal/gateway), entrypoint [`cmd/gateway/`](.).

## Routes

| Route | Behavior |
|---|---|
| `POST /wake` on `$GATEWAY_PORT` (published on 443) | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `BLACKBERRY_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |
| `GET`/`POST /settings` on `$GATEWAY_PORT` (published on 443) | Requires the identity header. Reads, or sets from form fields `wake`/`suspend` (`true`/`false`), the auto-wake and auto-suspend switches; returns `{"wake": ..., "suspend": ...}`. Logs `settings_changed`. Off auto-wake makes `wake` routes 502 while asleep; off auto-suspend keeps a pending auto-wake for when it's back on. |
| Each stack route, at its own hostname or under its `path` on 443 | Reverse-proxies to `upstream`, waking blackberry first if the route has `wake`. Logs `autowake_*` and `autosuspend_*`. |

## Routes file

Generated, not hand-written. `task gateway` runs [`cmd/routes`](../routes/main.go),
which reads the top-level `x-gateway` block of every `$STACKS_DIR/*/compose.yml`
(the [deploys](https://github.com/nikbogman/deploys) repo, default `../deploys`)
through `docker compose config`, validates the result, and ships it to
`/etc/gateway/routes.json`. A stack without `x-gateway` isn't routed.
`go run ./cmd/routes ../deploys` prints it without deploying.

| Field | Meaning |
|---|---|
| `idle_minutes` | Quiet time before gateway suspends blackberry. Top level: the default, 30 (`cmd/routes -idle-minutes`). Per route: an override, shorter for SSD-only stacks (popcorn), so the HDD only stays spun up for stacks that use it (immich). A route's use keeps blackberry up for at least its own time. |
| `name` | The Compose project's `name`. Label for logs, and the route's own tailnet hostname: `https://<name>.<tailnet>.ts.net` on 443, from an in-process tsnet node. A lowercase DNS label, unique. |
| `service` | `x-gateway` only. The service whose first published port is the upstream; optional when the stack has one service. |
| `port` | Only for `path` routes: the loopback listen port `tailscale serve` forwards to. Unique, never 443. |
| `path` | Optional. Publishes the route at `https://raspberry.<tailnet>.ts.net<path>` on 443 (`tailscale serve --set-path`) instead of at its own hostname. Looks like `/name`. Tailscale strips the prefix, so the stack sees `/`. Only for stacks that work behind a prefix (APIs, not web UIs). Can't be `public`. |
| `upstream` | Not set by hand: `http://<host_ip>:<published>` of `service`'s first port. That port must be bound to blackberry's LAN IP, never `0.0.0.0`. |
| `public` | `true` publishes the route's hostname on Funnel, with no identity check. `false` accepts tailnet callers only; the route's tsnet node looks the caller up and sets the identity header. |
| `wake` | `true` sends WoL and holds the request while blackberry is asleep. `false` returns 502 while it's asleep. |

## Environment

Read by [`internal/gateway/config.go`](../../internal/gateway/config.go), the
source of truth for names and defaults. Set in the repo root `.env` on the dev
machine; `task gateway` passes them to the unit. Bold variables are required.

| Variable | Purpose |
|---|---|
| **`BLACKBERRY_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`HOSTD_URL`** | hostd's tailnet URL, as raspberry reaches it. |
| **`UI_ORIGIN`** | The UI's Cloudflare Pages origin, allowed by CORS on `/wake`. Unset disables CORS (required by the deploy). |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `GATEWAY_PORT` | `/wake` listen port (default `5000`). The deploy pins it, since `tailscale serve` must point at it. |
| `GATEWAY_HOST` | `/wake` bind address (default `127.0.0.1`). Must be loopback or tailnet. The deploy never sets it. |
| `GATEWAY_ROUTES` | Routes file path (default `/etc/gateway/routes.json`). The deploy never sets it. |
| `GATEWAY_TSNET_DIR` | Where each hostname route keeps its tsnet node state (default `/var/lib/gateway/tsnet`). Losing it makes every route re-join as a new device. |
| `TS_AUTHKEY` | Read by tsnet when a route's node first joins the tailnet; `task gateway` passes `TAILSCALE_AUTH_KEY`. Use a reusable key, since each route is its own node. Not needed after the first start. |

## Deployment

`task gateway` ships the routes file, cross-compiles for the Pi Zero W, ships
the binary and a systemd unit (the device never runs a Go toolchain), and runs
`tailscale serve` for `$GATEWAY_PORT` on 443. At startup gateway starts a tsnet
node per hostname route and runs `tailscale serve` for each path route. Funnel
must be enabled in the tailnet policy for public routes. See
[`DEPLOY.md`](../../DEPLOY.md).

Replacing the old waker and gate units on raspberry, once:
`task cleanup DEVICE=RASPBERRY OLD=waker`, `task cleanup DEVICE=RASPBERRY OLD=gate`,
then `task gateway`.
