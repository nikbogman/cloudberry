# Edge

The public entry point, hosted on Railway. Serves the [UI](../../ui/README.md) and forwards
to [gateway](../gateway/README.md) and [hostd](../hostd/README.md) over the
tailnet, so the browser only ever talks to one origin.

**No authentication.** Anyone with the URL can wake and suspend blackberry. Deliberate and temporary — auth will be added later.

Code: [`internal/edge/`](../../internal/edge),
entrypoint [`cmd/edge/`](.).

## Routes

| Route | Forwards to |
|---|---|
| `POST /api/gateway/wake` | gateway `POST /wake` |
| `GET /api/hostd/health` | hostd `GET /health` |
| `POST /api/hostd/suspend` | hostd `POST /suspend` |
| `/ui/*` | The UI's static files, `//go:embed`ded from `ui/`. |
| `GET /` | Redirects to `/ui/`. |

Prefixes are stripped before forwarding. Any incoming `Tailscale-User-Login`
is dropped; `tailscale serve` on the far side sets it from Edge's own node.

## Tailnet access

Edge joins the tailnet as its own node (`edge`) via
[`tsnet`](https://pkg.go.dev/tailscale.com/tsnet), only to dial out. It listens
on Railway's public port, not on the tailnet.

`TS_AUTHKEY` **must be user-owned (untagged)**: gateway and hostd 401
without the Identity header, and `tailscale serve` only injects it for
user-owned nodes. Every Edge request is logged under that user.

## Logging

Every request ships one event to Grafana Cloud Loki (labels `app="edge"`,
`event_type="request"`, `outcome` = status class like `2xx`), with method, path,
status, duration and client IP (`X-Forwarded-For`) in the line. Code:
[`accesslog.go`](../../internal/edge/accesslog.go).

## Runtime environment

Read by [`internal/edge/config.go`](../../internal/edge/config.go). Bold variables
are required.

| Variable | Purpose |
|---|---|
| **`GATEWAY_URL`** | gateway origin, e.g. `https://raspberry.<tailnet>.ts.net`. Was `WAKER_URL`. |
| **`HOSTD_URL`** | hostd origin. |
| **`TS_STATE_DIR`** | tsnet node state. The image defaults it to `/data/tsnet`. |
| **`GRAFANA_CLOUD_LOKI_URL`**, **`_USER`**, **`_API_KEY`** | Same trio as [gateway](../gateway/README.md#environment). |
| `TS_AUTHKEY` | Needed on first start (or after state is lost). |
| `TS_DEBUG_MTU` | Tailnet MTU. The image sets `1200`: at tsnet's default 1280, replies from the homelab to Railway are lost and every tailnet request hangs. |
| `PORT` | Listen port (default `8080`; Railway sets it). |

## Deployment

Railway builds [`deployments/edge/Dockerfile`](../../deployments/edge/Dockerfile)
with the repo root as build context (service settings: Root Directory `/`,
Dockerfile Path `deployments/edge/Dockerfile`). Mount a
volume at `/data` — without it tsnet state is lost on every deploy and the node re-registers under a new name.

## Development

Set the required variables above, plus `TS_AUTHKEY` and a local
`TS_STATE_DIR`, then `go run ./cmd/edge`.
