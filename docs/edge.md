# Edge

The public entry point, hosted on Railway. Serves the [UI](ui.md) and forwards
to the [Waker API](waker.md), the [Sleeper API](sleeper.md) and Caddy-fronted
workloads over the tailnet, so the browser only ever talks to one origin.

**No authentication.** Anyone with the URL can wake, suspend and reach every
workload. Deliberate and temporary — auth will be added later.

Code: [`internal/edge/`](../internal/edge),
entrypoint [`cmd/edge-api/`](../cmd/edge-api).

## Routes

| Route | Forwards to |
|---|---|
| `POST /api/waker/wake` | Waker API `POST /wake` |
| `GET /api/sleeper/health` | Sleeper API `GET /health` |
| `POST /api/sleeper/suspend` | Sleeper API `POST /suspend` |
| `/proxy/<stack>/*` | `https://<stack>.$TAILNET_DOMAIN/*` — the stack's Caddy site. `<stack>` must match `^[a-z0-9-]+$`, else 400. |
| `/ui/*` | The UI's static files, `//go:embed`ded from `ui/`. |
| `GET /` | Redirects to `/ui/`. |

Prefixes are stripped before forwarding. Any incoming `Tailscale-User-Login`
is dropped; `tailscale serve` on the far side sets it from Edge's own node.

Known limitation: apps that assume they live at `/` may break under the
`/proxy/<stack>/` prefix.

## Tailnet access

Edge joins the tailnet as its own node (`edge`) via
[`tsnet`](https://pkg.go.dev/tailscale.com/tsnet), only to dial out. It listens
on Railway's public port, not on the tailnet.

`TS_AUTHKEY` **must be user-owned (untagged)**: the Waker and Sleeper APIs 401
without the Identity header, and `tailscale serve` only injects it for
user-owned nodes. Every Edge request is logged under that user.

## Logging

Every request ships one event to Grafana Cloud Loki (labels `app="edge"`,
`event_type="request"`, `outcome` = status class like `2xx`), with method, path,
status, duration and client IP (`X-Forwarded-For`) in the line. Code:
[`accesslog.go`](../internal/edge/accesslog.go).

## Runtime environment

Read by [`internal/edge/config.go`](../internal/edge/config.go). Bold variables
are required.

| Variable | Purpose |
|---|---|
| **`WAKER_API_URL`** | Waker API origin, e.g. `https://<waker>.<tailnet>.ts.net`. |
| **`SLEEPER_API_URL`** | Sleeper API origin. |
| **`TAILNET_DOMAIN`** | e.g. `<tailnet>.ts.net`; stack hosts are `<stack>.$TAILNET_DOMAIN`. |
| **`TS_STATE_DIR`** | tsnet node state. The image defaults it to `/data/tsnet`. |
| **`GRAFANA_CLOUD_LOKI_URL`**, **`_USER`**, **`_API_KEY`** | Grafana Cloud Loki push credentials, same as the Waker's. |
| `TS_AUTHKEY` | Needed on first start (or after state is lost). |
| `TS_DEBUG_MTU` | Tailnet MTU. The image sets `1200`: at tsnet's default 1280, replies from the homelab to Railway are lost and every tailnet request hangs. |
| `PORT` | Listen port (default `8080`; Railway sets it). |

## Deployment

Railway builds [`deployments/edge-api/Dockerfile`](../deployments/edge-api/Dockerfile)
with the repo root as build context (`RAILWAY_DOCKERFILE_PATH`). Mount a
volume at `/data` — without it tsnet state is lost on every deploy and the node re-registers under a new name.

## Development

```sh
TS_AUTHKEY=tskey-auth-xxx TS_STATE_DIR=/tmp/edge \
WAKER_API_URL=https://<waker>.<tailnet>.ts.net \
SLEEPER_API_URL=https://<sleeper>.<tailnet>.ts.net \
TAILNET_DOMAIN=<tailnet>.ts.net \
GRAFANA_CLOUD_LOKI_URL=... GRAFANA_CLOUD_LOKI_USER=... GRAFANA_CLOUD_LOKI_API_KEY=... \
go run ./cmd/edge-api
```
