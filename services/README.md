# Control-plane services (Go)

One Go module (`go.mod` in this directory, `github.com/nikbogman/homelab/services`) covering the Pi API, Server API, and the `tailnet`/`eventlog` libraries they share. See [ADR-0015](../docs/adr/0015-pi-api-and-server-api-rewritten-in-go-built-off-device.md) for why this is Go rather than the original Flask apps, and why it's one module rather than three.

```
cmd/pi-api/main.go        thin entrypoint: env → wiring → ListenAndServe
cmd/server-api/main.go    same, for server-api

internal/tailnet/         shared: tailnet identity-header auth, bind-safety
internal/eventlog/        shared: Grafana Cloud event logging
internal/piapi/           Pi API's own logic (the /wake handler, Wake-on-LAN packet building)
internal/serverapi/       Server API's own logic (CORS, reachability tracking, /health, /suspend, suspend command)
```

## Development

```sh
go vet ./...
go test ./...
```

Run a service locally with its own env vars (see each section below), e.g.:

```sh
SERVER_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
go run ./cmd/pi-api
```

## Pi API

Sends a Wake-on-LAN magic packet to the main server on request. Runs on the Pi Zero, path-routed under the same `tailscale serve` app as the [UI](ui) — same-origin, so no CORS entry is needed.

`POST /wake` — requires the `Tailscale-User-Login` identity header, or a caller on `127.0.0.1` ([ADR-0004](../docs/adr/0004-tailnet-membership-authorization.md)). Sends a WoL magic packet to the configured `SERVER_MAC_ADDRESS`, logs a `wake_requested`/`wake_succeeded`/`wake_failed` event to Grafana Cloud ([ADR-0014](../docs/adr/0014-events-shipped-directly-to-grafana-cloud.md)), and returns `{"wake": "succeeded"}` (200) or `{"wake": "failed"}` (500).

This is one of two wake triggers in the system — the other is the [Pi proxy](pi-proxy), which calls this endpoint automatically (via the loopback path above) on any proxied request while the server is asleep ([ADR-0012](../docs/adr/0012-auto-wake-proxy-calls-control-pi-api.md), superseding [ADR-0005](../docs/adr/0005-dual-wake-paths.md)'s original independent-paths design).

Requires the Pi and the main server to share an L2 broadcast domain ([ADR-0003](../docs/adr/0003-wol-same-broadcast-domain.md)) — WoL packets don't route across subnets.

| Variable | Required | Purpose |
|---|---|---|
| `SERVER_MAC_ADDRESS` | yes | MAC address the magic packet targets. Validated at startup — a malformed value fails fast, not mid-request. |
| `GRAFANA_CLOUD_LOKI_URL` | yes | Grafana Cloud's Loki push endpoint for your stack ([ADR-0014](../docs/adr/0014-events-shipped-directly-to-grafana-cloud.md)) — events are POSTed here directly, no local collector in between. |
| `GRAFANA_CLOUD_LOKI_USER` | yes | Grafana Cloud Loki basic-auth username (the stack's numeric instance/user ID). |
| `GRAFANA_CLOUD_LOKI_API_KEY` | yes | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `PI_API_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address ([`tailnet.AssertTailnetOnlyBind`](internal/tailnet/bindsafety.go)) — the process refuses to start if this would expose it off-tailnet. |
| `PI_API_PORT` | no (default `5000`) | Port to listen on. |

Deployed as a systemd unit on the Pi Zero, not Docker ([ADR-0007](../docs/adr/0007-systemd-not-docker-for-control-apis.md)). The binary is cross-compiled for the Pi Zero W on the dev machine and shipped — it never runs a Go toolchain itself ([ADR-0013](../docs/adr/0013-caddy-binary-built-off-device-by-pyinfra.md), [ADR-0015](../docs/adr/0015-pi-api-and-server-api-rewritten-in-go-built-off-device.md)). Automated by pyinfra's `deploy_pi_api.py` (`../pyinfra/deploy_pi_api.py`) — see [`pyinfra/README.md`](../pyinfra/README.md).

## Server API

Exposes a Reachable health check and the Suspend action. Runs on the main server behind its own `tailscale serve` instance — a distinct origin from the [UI](ui), hence the CORS allow-list ([ADR-0001](../docs/adr/0001-direct-browser-to-api-no-relay.md)).

Both endpoints require the `Tailscale-User-Login` identity header ([ADR-0004](../docs/adr/0004-tailnet-membership-authorization.md)).

- `GET /health` — returns `{"reachable": true}` (200). The UI polls this to decide whether the server is [Reachable](../CONTEXT.md) and which of Wake/Suspend to offer. The first time a process observes itself reachable, it logs a `reachability_changed` event to Grafana Cloud — see `reachabilityTracker` in [`internal/serverapi/handler.go`](internal/serverapi/handler.go) for why this can only ever fire once per process (it can't witness going *un*reachable, since it's asleep while that's true).
- `POST /suspend` — runs the configured suspend command (`systemctl suspend` by default, see [`internal/serverapi/suspend.go`](internal/serverapi/suspend.go)), logs `suspend_requested`/`suspend_succeeded`/`suspend_failed` to Grafana Cloud, and returns `{"suspend": "succeeded"}` (200) or `{"suspend": "failed"}` (500).

Suspend-to-RAM is the only sleep state this app can trigger — no code path anywhere in it can cause a full shutdown (ACPI S5), and the suspend command is intentionally not configurable via environment variable, so a misconfiguration can't reintroduce one ([ADR-0002](../docs/adr/0002-suspend-only-no-shutdown.md)).

| Variable | Required | Purpose |
|---|---|---|
| `UI_ORIGIN` | yes | Origin allowed by the CORS policy — the UI's origin. |
| `GRAFANA_CLOUD_LOKI_URL` | yes | Grafana Cloud's Loki push endpoint for your stack. |
| `GRAFANA_CLOUD_LOKI_USER` | yes | Grafana Cloud Loki basic-auth username. |
| `GRAFANA_CLOUD_LOKI_API_KEY` | yes | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `SERVER_API_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address — the process refuses to start if this would expose it off-tailnet. |
| `SERVER_API_PORT` | no (default `5000`) | Port to listen on. |

Deployed as a systemd unit on the main server, not Docker — chosen so this app stays controllable even while Docker itself is being redeployed ([ADR-0007](../docs/adr/0007-systemd-not-docker-for-control-apis.md)). Built off-device and shipped as a binary, same as the Pi API ([ADR-0015](../docs/adr/0015-pi-api-and-server-api-rewritten-in-go-built-off-device.md)). Automated by pyinfra's `deploy_server_api.py` (`../pyinfra/deploy_server_api.py`).

## `internal/tailnet`

Shared tailnet identity-auth and bind-safety helpers, not a standalone service — pulled in as a package by both binaries within this module.

- **`auth.go`** — `RequireTailnetIdentity`, an HTTP middleware that rejects any request missing the `Tailscale-User-Login` header (the [Identity header](../CONTEXT.md)) with a 401, and `GetCallerIdentity` to read it back inside a handler. Per [ADR-0004](../docs/adr/0004-tailnet-membership-authorization.md), presence of the header is the entire authorization check — no allow-list of specific identities.
- **`bindsafety.go`** — `AssertTailnetOnlyBind(host)`, called at handler construction, returns a `*BindOffTailnetError` unless `host` is loopback or within Tailscale's own address ranges. A structural backstop for the "never reachable off-tailnet" requirement, independent of whatever fronts the app.

## `internal/eventlog`

Ships structured wake/suspend/reachability events straight to Grafana Cloud's Loki push endpoint ([ADR-0014](../docs/adr/0014-events-shipped-directly-to-grafana-cloud.md)) — kept separate from `tailnet` since it's not a tailnet concept, just a logger both apps happen to share.

- **`grafanacloud.go`** — `GrafanaCloudLogger.SendEvent(...)`. Both apps are stateless, so Grafana Cloud is the only place this history lives. Transport failures are logged and swallowed — a logging outage must never block the caller's actual action.
