# gateway

Runs on raspberry, the always-on Pi Zero W. It sends Wake-on-LAN to
blackberry in two cases:

- **The UI's Wake.** The browser calls `POST /wake` directly, over the tailnet.
- **A request to a stack route flagged `wake`** in the [routes file](#routes-file).
  gateway holds the request until the route's upstream accepts a TCP connection
  on the LAN, then reverse-proxies it. Once every route has been quiet for its
  `idle_minutes` it calls hostd `/suspend`, but only if it auto-woke blackberry.

gateway also probes blackberry every 15s, dialing the first route's upstream
(every 250ms while a wake it sent is pending), so it knows when blackberry
woke or went to sleep whoever caused it, and how long a wake took. That's
`GET /status`.

Both are switchable at runtime from the UI (`/settings`). The switches,
whether gateway did the last wake, and the status history are saved to
`$GATEWAY_STATE_FILE`, so the
restarts that updates cause keep them.

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
| `GET /status` | Requires the identity header. Everything the UI shows: `awake` (null until the first probe), `since`, `woken_by` (a route name or a login; empty if gateway didn't send the wake), `last_wake_seconds`, `sleeps_at` and `idle_window_seconds` (only when gateway did the waking and auto-suspend is on), `wake_pending`, the switches, each route's `url` and `last_request`, and the wakes and suspends of the last 48h (`events`). |
| Each stack route, at its own hostname or under its `path` on 443 | Reverse-proxies to `upstream`, waking blackberry first if the route has `wake`. Logs `autowake_*` and `autosuspend_*`. |

## Routes file

Generated, not hand-written, from the top-level `x-route` block of every
`*/compose.yml` in the [deploys](https://github.com/nikbogman/deploys) repo,
through `docker compose config` ([`internal/routes`](../../internal/routes)).
A stack without `x-route` isn't routed. Two ways in, same result:

- **Automatically.** hostd serves it at `GET /routes` from its clone of
  deploys. gateway polls that every `$UPDATE_MINUTES`; hostd only answers while
  blackberry is awake, which is also the only time stacks change. A different,
  valid file is written to `/etc/gateway/routes.json` and gateway restarts on it
  (`routes_updated`); an invalid one is ignored (`routes_rejected`).
- **`task gateway`**, from a dev machine checkout (`$STACKS_DIR`, default
  `../deploys`), via [`cmd/routes`](../routes/main.go).
  `go run ./cmd/routes ../deploys` prints it without deploying.

A path route removed from the file is gone on the next start.

| Field | Meaning |
|---|---|
| `idle_minutes` | Quiet time before gateway suspends blackberry. Top level: the default, 30 (`cmd/routes -idle-minutes`). Per route: an override, shorter for SSD-only stacks, so the HDD only stays spun up for stacks that use it (immich). A route's use keeps blackberry up for at least its own time. |
| `name` | The Compose project's `name`. Label for logs, and the route's own tailnet hostname: `https://<name>.<tailnet>.ts.net` on 443, from an in-process tsnet node. A lowercase DNS label, unique. |
| `service` | `x-route` only. The service whose first published port is the upstream; optional when the stack has one service. |
| `path` | Optional. Serves the route at `https://raspberry.<tailnet>.ts.net<path>/`, on gateway's own node, instead of at its own hostname. Looks like `/name`; not `/wake` or `/settings`. gateway strips the prefix, so the stack sees `/`. Only for stacks that work behind a prefix (APIs, not web UIs). Can't be `public`. |
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
| `GATEWAY_HOSTNAME` | gateway's own tsnet node (default `raspberry`): `https://<name>.<tailnet>.ts.net` serves `/wake`, `/settings` and path routes, and carries gateway's calls to hostd. No route may use the name. |
| `GATEWAY_PORT` | Loopback listen port (default `5000`), for `/wake` from a shell on raspberry. |
| `GATEWAY_HOST` | `/wake` bind address (default `127.0.0.1`). Must be loopback or tailnet. The deploy never sets it. |
| `GATEWAY_ROUTES` | Routes file path (default `/etc/gateway/routes.json`). The deploy never sets it. |
| `GATEWAY_TSNET_DIR` | Where each hostname route keeps its tsnet node state (default `/var/lib/gateway/tsnet`). Losing it makes every route re-join as a new device. |
| `GATEWAY_STATE_FILE` | Saved switches and auto-wake flag (default `/var/lib/gateway/state.json`). The deploy never sets it. |
| `UPDATE_MINUTES` | How often to poll hostd's routes and GitHub's latest release (default `2`). |
| `RELEASE_URL` | Where releases come from (default `https://github.com/nikbogman/metal`). Empty turns binary updates off. |
| `TS_AUTHKEY` | Read by tsnet when a route's node first joins the tailnet; `task gateway` passes `TAILSCALE_AUTH_KEY`. Use a reusable key, since each route is its own node. Not needed after the first start. |

## Updates

Every `$UPDATE_MINUTES`, besides the routes file above, gateway checks the
latest [release](https://github.com/nikbogman/metal/releases) (the
`/releases/latest` redirect, not the rate-limited API). A newer one is
downloaded, checked against `SHA256SUMS`, and swapped in by rename, keeping
the old binary as `gateway.prev` (`binary_updated`). Either kind of update
waits up to a minute for held requests and running wakes, then exits;
systemd (`Restart=always`) starts the new one. Failures log `update_failed`,
once per distinct error.

Only release builds update themselves: their version is a `vX.Y.Z` tag. A
binary built from anything else (`task gateway` on an untagged or dirty
checkout) stays as deployed, which is how to pin one. If a new binary fails
three starts in a minute, `gateway-rollback.service` moves it to `gateway.bad`
and puts `gateway.prev` back; that release isn't installed again.

## Deployment

`task gateway` ships the routes file, cross-compiles for the Pi Zero W, and
ships the binary, a systemd unit and its rollback unit (the device never runs a
Go toolchain). raspberry runs no tailscaled: at startup gateway brings up its
own tsnet node (`$GATEWAY_HOSTNAME`) and one per hostname route, all under
`$GATEWAY_TSNET_DIR`. Funnel must be enabled in the tailnet policy for public
routes. See
[`DEPLOY.md`](../../DEPLOY.md).

Replacing the old waker and gate units on raspberry, once:
`task cleanup DEVICE=RASPBERRY OLD=waker`, `task cleanup DEVICE=RASPBERRY OLD=gate`,
then `task gateway`.
