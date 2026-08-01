# Control-plane services 

The backend for the homelab control system: two services — the Gateway API (Pi Zero, always on) and the Compute API (the workload host that sleeps) — that let the UI wake, monitor, and suspend Compute from anywhere on the tailnet. One Go module (`go.mod` in this directory, `github.com/nikbogman/homelab/control-plane`) covers both binaries. See [`ARCHITECTURE.md`](../ARCHITECTURE.md) for why both are Go and share code directly.

```
control-plane/
├── cmd/               entrypoints
│   ├── gateway/
│   └── compute-api/
└── internal/
    ├── tailnet/       shared: tailnet identity-header auth, bind-safety
    ├── eventlog/      shared: Grafana Cloud event logging
    ├── httpresponse/  shared: JSON response writing, response pass-through
    ├── env/           shared: env-var lookup helpers used by cmd/* entrypoints
    ├── gateway/       Gateway API's own logic (static file serving, the /wake handler and WoL packet building, the /server* proxy with auto-wake)
    └── compute/       Compute API's own logic (CORS, reachability tracking, /health, /suspend, suspend command)
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
COMPUTE_HOST=main-server.tailnet \
COMPUTE_PROXY_PORT=8081 \
go run ./cmd/gateway
```

## Gateway API

Serves the UI's static files, sends a Wake-on-LAN magic packet to the compute host on request, and reverse-proxies workload traffic to the compute host — one process doing all three jobs. Runs on the Pi Zero, same-origin with the [UI](ui) it serves — no CORS entry is needed.

`POST /wake` — requires the `Tailscale-User-Login` identity header, or a caller on `127.0.0.1`. Sends a WoL magic packet to the configured `COMPUTE_MAC_ADDRESS`, logs a `wake_requested`/`wake_succeeded`/`wake_failed` event to Grafana Cloud, and returns `{"wake": "succeeded"}` (200) or `{"wake": "failed"}` (500).

`/server/*` — reverse-proxies to `http://{COMPUTE_HOST}:{COMPUTE_PROXY_PORT}`, path-stripped, as a single blind upstream — which path reaches which Docker service (Immich, AI agents, etc.) is entirely that host's own reverse proxy's concern. A genuine HTTP response from a live backend (including a real 502) passes straight through untouched. A Go-level transport failure (dial refused, timeout) instead triggers a wake (throttled to once per 90s, since there's exactly one compute upstream) and retries the request every 1s for up to 60s, writing through the first successful response — this is one of two wake triggers in the system, the other being the deliberate `/wake` above.

`/` and everything else — serves the UI's static build, `//go:embed`ded into this binary at build time (no separate on-device path, no SPA/`try_files` fallback: an unknown path 404s).

Requires the Gateway and the compute host to share an L2 broadcast domain — WoL packets don't route across subnets.

Bold variables are required.

| Variable | Purpose |
|---|---|
| **`COMPUTE_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup — a malformed value fails fast, not mid-request. |
| **`COMPUTE_HOST`** | Host the `/server*` route forwards to. |
| **`COMPUTE_PROXY_PORT`** | Port on `COMPUTE_HOST` that `/server*` forwards to (path stripped); no default since the server-side proxy it points at doesn't exist yet. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint for your stack — events are POSTed here directly, no local collector in between. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username (the stack's numeric instance/user ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `GATEWAY_HOST` | Bind address (default `127.0.0.1`). Asserted at startup to be loopback or a tailnet address ([`tailnet.AssertTailnetOnlyBind`](internal/tailnet/bindsafety.go)) — the process refuses to start if this would expose it off-tailnet. |
| `GATEWAY_PORT` | Port to listen on (default `5000`). |

Deployed as a systemd unit on the Pi Zero, not Docker. The binary is cross-compiled for the Pi Zero W on the dev machine and shipped — it never runs a Go toolchain itself. Automated by pyinfra's `deploy_gateway.py` (`../provisioning/deploy_gateway.py`) — see [`provisioning/README.md`](../provisioning/README.md).

## Compute API

Exposes a Reachable health check and the Suspend action. Runs on the compute host behind its own `tailscale serve` instance — a distinct origin from the [UI](ui), hence the CORS allow-list.

Both endpoints require the `Tailscale-User-Login` identity header.

- `GET /health` — returns `{"reachable": true}` (200). The UI polls this to decide whether the compute host is [Reachable](../CONTEXT.md) and which of Wake/Suspend to offer. The first time a process observes itself reachable, it logs a `reachability_changed` event to Grafana Cloud — see `reachabilityTracker` in [`internal/compute/handler.go`](internal/compute/handler.go) for why this can only ever fire once per process (it can't witness going *un*reachable, since it's asleep while that's true).
- `POST /suspend` — runs the configured suspend command (`systemctl suspend` by default, see [`internal/compute/suspend.go`](internal/compute/suspend.go)), logs `suspend_requested`/`suspend_succeeded`/`suspend_failed` to Grafana Cloud, and returns `{"suspend": "succeeded"}` (200) or `{"suspend": "failed"}` (500).

Suspend-to-RAM is the only sleep state this app can trigger — no code path anywhere in it can cause a full shutdown (ACPI S5), and the suspend command is intentionally not configurable via environment variable, so a misconfiguration can't reintroduce one.

Bold variables are required.

| Variable | Purpose |
|---|---|
| **`UI_ORIGIN`** | Origin allowed by the CORS policy — the UI's origin. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint for your stack. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username. |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `COMPUTE_API_HOST` | Bind address (default `127.0.0.1`). Asserted at startup to be loopback or a tailnet address — the process refuses to start if this would expose it off-tailnet. |
| `COMPUTE_API_PORT` | Port to listen on (default `5000`). |

Deployed as a systemd unit on the compute host, not Docker — chosen so this app stays controllable even while Docker itself is being redeployed. Built off-device and shipped as a binary, same as the Gateway API. Automated by pyinfra's `deploy_compute_api.py` (`../provisioning/deploy_compute_api.py`).

## `internal/tailnet`

Shared tailnet identity-auth and bind-safety helpers, not a standalone service — pulled in as a package by both binaries within this module.

- **`auth.go`** — `RequireTailnetIdentity`, an HTTP middleware that rejects any request missing the `Tailscale-User-Login` header (the [Identity header](../CONTEXT.md)) with a 401, and `GetCallerIdentity` to read it back inside a handler. Presence of the header is the entire authorization check — no allow-list of specific identities.
- **`bindsafety.go`** — `AssertTailnetOnlyBind(host)`, called at handler construction, returns a `*BindOffTailnetError` unless `host` is loopback or within Tailscale's own address ranges. A structural backstop for the "never reachable off-tailnet" requirement, independent of whatever fronts the app.

## `internal/eventlog`

Ships structured wake/suspend/reachability events straight to Grafana Cloud's Loki push endpoint — kept separate from `tailnet` since it's not a tailnet concept, just a logger both apps happen to share.

- **`grafanacloud.go`** — `GrafanaCloudLogger.SendEvent(...)`. Both apps are stateless, so Grafana Cloud is the only place this history lives. Transport failures are logged and swallowed — a logging outage must never block the caller's actual action.
