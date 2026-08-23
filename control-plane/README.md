# Control-plane services

Backend for the homelab control system: the Gateway API (Pi Zero, always on) and the Compute API (the host that sleeps). One Go module (`go.mod`, `github.com/nikbogman/homelab/control-plane`) covers both binaries. See [`DESIGN.md`](../DESIGN.md) for why.

```
control-plane/
├── cmd/               entrypoints
│   ├── gateway/
│   └── compute-api/
└── internal/
    ├── tailnet/       shared: identity-header auth, bind-safety
    ├── eventlog/      shared: Grafana Cloud event logging
    ├── httpresponse/  shared: JSON response writing, response pass-through
    ├── env/           shared: env-var lookup helpers
    ├── gateway/       Gateway API logic (static serving, /wake, /server* proxy + auto-wake)
    └── compute/       Compute API logic (CORS, reachability tracking, /health, /suspend, /hold, /status, container proxy, idle watcher)
```

## Development

```sh
go vet ./...
go test ./...
```

Run a service locally with its env vars (see below), e.g.:

```sh
COMPUTE_MAC_ADDRESS=AA:BB:CC:DD:EE:FF \
GRAFANA_CLOUD_LOKI_URL=https://logs-prod-000.grafana.net/loki/api/v1/push \
GRAFANA_CLOUD_LOKI_USER=123456 GRAFANA_CLOUD_LOKI_API_KEY=glc_xxx \
COMPUTE_HOST=main-server.tailnet \
COMPUTE_PROXY_PORT=8081 \
go run ./cmd/gateway
```

## Gateway API

Serves the UI's static files, sends WoL to Compute, and reverse-proxies workload traffic — one process, all three jobs. Runs on the Pi Zero, same-origin with the [UI](ui) — no CORS entry needed.

| Route | Behavior |
|---|---|
| `POST /wake` | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `COMPUTE_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |
| `/server/*` | Reverse-proxies to `http://{COMPUTE_HOST}:{COMPUTE_PROXY_PORT}`, path-stripped. A real backend response (even 502) passes through untouched; a transport failure triggers a throttled wake and retries for up to 60s. |
| `/` (everything else) | Serves the UI's static build, `//go:embed`ded at build time. Unknown paths 404 (no SPA fallback). |

Requires the Gateway and compute host to share an L2 broadcast domain — WoL doesn't route across subnets.

Bold variables are required.

| Variable | Purpose |
|---|---|
| **`COMPUTE_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`COMPUTE_HOST`** | Host the `/server*` route forwards to. |
| **`COMPUTE_PROXY_PORT`** | Port on `COMPUTE_HOST` that `/server*` forwards to; no default. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `GATEWAY_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet ([`tailnet.AssertTailnetOnlyBind`](internal/tailnet/bindsafety.go)). |
| `GATEWAY_PORT` | Listen port (default `5000`). |

Deployed as a systemd unit on the Pi Zero, cross-compiled and shipped from the dev machine (never runs a Go toolchain itself). Automated by [`deploy/deploy_gateway.py`](../deploy/deploy_gateway.py) — see [`deploy/README.md`](../deploy/README.md).

## Compute API

Exposes a Reachable health check, the Suspend action, and its own reverse proxy to workload Docker containers, behind its own `tailscale serve` instance — a distinct origin from the [UI](ui), hence the CORS allow-list.

| Route | Behavior |
|---|---|
| `GET /health` | Requires the identity header. Returns `{"reachable": true}`. First time reachable, logs `reachability_changed` (see `reachabilityTracker` in [`internal/compute/handler.go`](internal/compute/handler.go) — it can never observe going unreachable, since the process is asleep whenever that's true). |
| `POST /suspend` | Requires the identity header. Runs the configured suspend command (`systemctl suspend` by default, see [`internal/compute/suspend.go`](internal/compute/suspend.go)), logs `suspend_requested`/`_succeeded`/`_failed`, returns `{"suspend": ...}`. |
| `POST /hold` | Requires the identity header or a `127.0.0.1` caller. Acquires or renews a fixed 30-minute hold that blocks automatic suspend, logs `hold_acquired`, returns `{"hold": "acquired"}` (see [`internal/compute/hold.go`](internal/compute/hold.go)). |
| `DELETE /hold` | Requires the identity header or a `127.0.0.1` caller. Releases an active hold, logs `hold_released`; a no-op (no event) if nothing was held, so a script can call it unconditionally on exit. |
| `GET /status` | Requires the identity header (no loopback exception — read diagnostic, not an automation target). Returns idle duration, hold state + remaining TTL, and whether dry-run mode is on. |
| `/{prefix}/*` (everything else) | Not identity-gated, mirroring the Gateway's `/server/` route. Reverse-proxies to whichever Docker container carries a `homelab.route={prefix}` label, path-stripped (see [`internal/compute/proxy.go`](internal/compute/proxy.go)). A container is only routable if it publishes a port to the host. A real backend response (even an error status) passes through untouched; an unmatched path 404s; a container-discovery failure 502s. Each successful proxy is timestamped, readable via `Handler.LastProxiedAt()` — an activity signal for the idle watcher below. |

Suspend-to-RAM only — the suspend command isn't configurable via env var, so misconfiguration can't reintroduce a full shutdown.

Container discovery talks to the local Docker daemon via the official Docker Go SDK ([`internal/compute/docker.go`](internal/compute/docker.go)), injected as the `ContainerRuntime` interface — same construction pattern as `Suspender`/`EventLogger`, faked in tests with no real daemon required.

**Idle watcher** ([`internal/compute/idlewatcher.go`](internal/compute/idlewatcher.go)) runs alongside the handler, polling once a minute. Compute counts as active if container CPU is above baseline, a request was proxied within the current window, or a Hold is active — container running-state alone is never the signal, since workload containers stay up regardless of use. After `COMPUTE_API_IDLE_TIMEOUT` with none of those true, it calls the same `Suspender.Suspend()` `POST /suspend` uses, logging `suspend_auto_triggered`/`_succeeded`/`_failed` instead of the manual event types. With `COMPUTE_API_AUTOSUSPEND_DRY_RUN` set, it runs the identical decision logic but logs `suspend_auto_dry_run` and keeps cycling instead of actually suspending — for validating the watcher live without disrupting the machine.

Bold variables are required.

| Variable | Purpose |
|---|---|
| **`UI_ORIGIN`** | Origin allowed by the CORS policy. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username. |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `COMPUTE_API_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet. |
| `COMPUTE_API_PORT` | Listen port (default `5000`). |
| `COMPUTE_API_IDLE_TIMEOUT` | Idle time before auto-suspend triggers (default `1h`). |
| `COMPUTE_API_CPU_BASELINE_PERCENT` | Container CPU% above which counts as activity (default `2`). |
| `COMPUTE_API_AUTOSUSPEND_DRY_RUN` | If `true`, the idle watcher logs instead of actually suspending (default `false`). |

Deployed as a systemd unit on the compute host (not Docker, so it stays controllable while Docker redeploys), built off-device same as the Gateway API. Automated by [`deploy/deploy_compute_api.py`](../deploy/deploy_compute_api.py).

## `internal/tailnet`

Shared identity-auth and bind-safety helpers, not a standalone service.

- **`auth.go`** — `RequireTailnetIdentity` middleware rejects requests missing the [Identity header](../CONTEXT.md) with a 401; `GetCallerIdentity` reads it back. Presence is the entire check — no allow-list.
- **`bindsafety.go`** — `AssertTailnetOnlyBind(host)`, called at handler construction, errors unless `host` is loopback or a Tailscale address.

## `internal/eventlog`

Ships structured wake/suspend/reachability events to Grafana Cloud's Loki push endpoint.

- **`grafanacloud.go`** — `GrafanaCloudLogger.SendEvent(...)`. Both apps are stateless, so Grafana Cloud is the only place this history lives. Transport failures are logged and swallowed.
