# Control-plane services (Go)

One Go module (`go.mod` in this directory, `github.com/nikbogman/homelab/services`) covering the Gateway API, Compute API, and the `tailnet`/`eventlog` libraries they share. See [ADR-0015](../docs/adr/0015-pi-api-and-server-api-rewritten-in-go-built-off-device.md) for why this is Go rather than the original Flask apps, and why it's one module rather than three.

```
cmd/gateway/main.go      thin entrypoint: env → wiring → ListenAndServe
cmd/compute-api/main.go   same, for compute-api

internal/tailnet/         shared: tailnet identity-header auth, bind-safety
internal/eventlog/        shared: Grafana Cloud event logging
internal/gateway/         Gateway API's own logic (static file serving, the /wake handler and WoL packet building, the /server* proxy with auto-wake)
internal/compute/         Compute API's own logic (CORS, reachability tracking, /health, /suspend, suspend command)
```

## Development

```sh
go vet ./...
go test ./...
```

Run a service locally with its own env vars (see each section below), e.g.:

```sh
COMPUTE_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
COMPUTE_HOST=main-server.tailnet COMPUTE_PROXY_PORT=8081 \
go run ./cmd/gateway
```

## Gateway API

Serves the UI's static files, sends a Wake-on-LAN magic packet to the compute host on request, and reverse-proxies workload traffic to the compute host — one process doing all three jobs ([ADR-0016](../docs/adr/0016-caddy-removed-gateway-absorbs-its-job.md), which removed Caddy from the Gateway and folded its job in here). Runs on the Pi Zero, same-origin with the [UI](ui) it serves — no CORS entry is needed.

`POST /wake` — requires the `Tailscale-User-Login` identity header, or a caller on `127.0.0.1` ([ADR-0004](../docs/adr/0004-tailnet-membership-authorization.md)). Sends a WoL magic packet to the configured `COMPUTE_MAC_ADDRESS`, logs a `wake_requested`/`wake_succeeded`/`wake_failed` event to Grafana Cloud ([ADR-0014](../docs/adr/0014-events-shipped-directly-to-grafana-cloud.md)), and returns `{"wake": "succeeded"}` (200) or `{"wake": "failed"}` (500).

`/server/*` — reverse-proxies to `http://{COMPUTE_HOST}:{COMPUTE_PROXY_PORT}`, path-stripped, as a single blind upstream ([ADR-0011](../docs/adr/0011-server-owns-workload-routing.md)) — which path reaches which Docker service (Immich, AI agents, etc.) is entirely that host's own reverse proxy's concern. A genuine HTTP response from a live backend (including a real 502) passes straight through untouched. A Go-level transport failure (dial refused, timeout) instead triggers a wake (throttled to once per 90s, since there's exactly one compute upstream) and retries the request every 1s for up to 60s, writing through the first successful response — this is one of two wake triggers in the system, the other being the deliberate `/wake` above ([ADR-0012](../docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md), superseding [ADR-0005](../docs/adr/0005-dual-wake-paths.md)'s original independent-paths design).

`/` and everything else — serves the UI's static build from `/srv/ui` (no SPA/`try_files` fallback: an unknown path 404s).

Requires the Gateway and the compute host to share an L2 broadcast domain ([ADR-0003](../docs/adr/0003-wol-same-broadcast-domain.md)) — WoL packets don't route across subnets.

| Variable | Required | Purpose |
|---|---|---|
| `COMPUTE_MAC_ADDRESS` | yes | MAC address the magic packet targets. Validated at startup — a malformed value fails fast, not mid-request. |
| `COMPUTE_HOST` | yes | Host the `/server*` route forwards to. |
| `COMPUTE_PROXY_PORT` | yes | Port on `COMPUTE_HOST` that `/server*` forwards to (path stripped) — [ADR-0011](../docs/adr/0011-server-owns-workload-routing.md); no default since the server-side proxy it points at doesn't exist yet. |
| `GRAFANA_CLOUD_LOKI_URL` | yes | Grafana Cloud's Loki push endpoint for your stack ([ADR-0014](../docs/adr/0014-events-shipped-directly-to-grafana-cloud.md)) — events are POSTed here directly, no local collector in between. |
| `GRAFANA_CLOUD_LOKI_USER` | yes | Grafana Cloud Loki basic-auth username (the stack's numeric instance/user ID). |
| `GRAFANA_CLOUD_LOKI_API_KEY` | yes | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `GATEWAY_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address ([`tailnet.AssertTailnetOnlyBind`](internal/tailnet/bindsafety.go)) — the process refuses to start if this would expose it off-tailnet. |
| `GATEWAY_PORT` | no (default `5000`) | Port to listen on. |

Deployed as a systemd unit on the Pi Zero, not Docker ([ADR-0007](../docs/adr/0007-systemd-not-docker-for-control-apis.md)). The binary is cross-compiled for the Pi Zero W on the dev machine and shipped — it never runs a Go toolchain itself ([ADR-0015](../docs/adr/0015-pi-api-and-server-api-rewritten-in-go-built-off-device.md)). Automated by pyinfra's `deploy_gateway.py` (`../pyinfra/deploy_gateway.py`) — see [`pyinfra/README.md`](../pyinfra/README.md).

## Compute API

Exposes a Reachable health check and the Suspend action. Runs on the compute host behind its own `tailscale serve` instance — a distinct origin from the [UI](ui), hence the CORS allow-list ([ADR-0001](../docs/adr/0001-direct-browser-to-api-no-relay.md)).

Both endpoints require the `Tailscale-User-Login` identity header ([ADR-0004](../docs/adr/0004-tailnet-membership-authorization.md)).

- `GET /health` — returns `{"reachable": true}` (200). The UI polls this to decide whether the compute host is [Reachable](../CONTEXT.md) and which of Wake/Suspend to offer. The first time a process observes itself reachable, it logs a `reachability_changed` event to Grafana Cloud — see `reachabilityTracker` in [`internal/compute/handler.go`](internal/compute/handler.go) for why this can only ever fire once per process (it can't witness going *un*reachable, since it's asleep while that's true).
- `POST /suspend` — runs the configured suspend command (`systemctl suspend` by default, see [`internal/compute/suspend.go`](internal/compute/suspend.go)), logs `suspend_requested`/`suspend_succeeded`/`suspend_failed` to Grafana Cloud, and returns `{"suspend": "succeeded"}` (200) or `{"suspend": "failed"}` (500).

Suspend-to-RAM is the only sleep state this app can trigger — no code path anywhere in it can cause a full shutdown (ACPI S5), and the suspend command is intentionally not configurable via environment variable, so a misconfiguration can't reintroduce one ([ADR-0002](../docs/adr/0002-suspend-only-no-shutdown.md)).

| Variable | Required | Purpose |
|---|---|---|
| `UI_ORIGIN` | yes | Origin allowed by the CORS policy — the UI's origin. |
| `GRAFANA_CLOUD_LOKI_URL` | yes | Grafana Cloud's Loki push endpoint for your stack. |
| `GRAFANA_CLOUD_LOKI_USER` | yes | Grafana Cloud Loki basic-auth username. |
| `GRAFANA_CLOUD_LOKI_API_KEY` | yes | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `COMPUTE_API_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address — the process refuses to start if this would expose it off-tailnet. |
| `COMPUTE_API_PORT` | no (default `5000`) | Port to listen on. |

Deployed as a systemd unit on the compute host, not Docker — chosen so this app stays controllable even while Docker itself is being redeployed ([ADR-0007](../docs/adr/0007-systemd-not-docker-for-control-apis.md)). Built off-device and shipped as a binary, same as the Gateway API ([ADR-0015](../docs/adr/0015-pi-api-and-server-api-rewritten-in-go-built-off-device.md)). Automated by pyinfra's `deploy_compute_api.py` (`../pyinfra/deploy_compute_api.py`).

## `internal/tailnet`

Shared tailnet identity-auth and bind-safety helpers, not a standalone service — pulled in as a package by both binaries within this module.

- **`auth.go`** — `RequireTailnetIdentity`, an HTTP middleware that rejects any request missing the `Tailscale-User-Login` header (the [Identity header](../CONTEXT.md)) with a 401, and `GetCallerIdentity` to read it back inside a handler. Per [ADR-0004](../docs/adr/0004-tailnet-membership-authorization.md), presence of the header is the entire authorization check — no allow-list of specific identities.
- **`bindsafety.go`** — `AssertTailnetOnlyBind(host)`, called at handler construction, returns a `*BindOffTailnetError` unless `host` is loopback or within Tailscale's own address ranges. A structural backstop for the "never reachable off-tailnet" requirement, independent of whatever fronts the app.

## `internal/eventlog`

Ships structured wake/suspend/reachability events straight to Grafana Cloud's Loki push endpoint ([ADR-0014](../docs/adr/0014-events-shipped-directly-to-grafana-cloud.md)) — kept separate from `tailnet` since it's not a tailnet concept, just a logger both apps happen to share.

- **`grafanacloud.go`** — `GrafanaCloudLogger.SendEvent(...)`. Both apps are stateless, so Grafana Cloud is the only place this history lives. Transport failures are logged and swallowed — a logging outage must never block the caller's actual action.
