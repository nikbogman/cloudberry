# Gateway

The always-on device (a Pi Zero W) and the Gateway API binary it runs. One
process does three jobs: serves the UI's static files, sends Wake-on-LAN to
Compute, and reverse-proxies workload traffic. Same-origin with the
[UI](ui.md), so no CORS entry is needed.

Vocabulary (Gateway, Wake, Identity header) is in [`CONTEXT.md`](../CONTEXT.md).

Code: [`internal/gateway/`](../internal/gateway),
entrypoint [`cmd/gateway-api/`](../cmd/gateway-api).

## Routes

| Route | Behavior |
|---|---|
| `POST /wake` | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `COMPUTE_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |
| `/server/*` | Reverse-proxies to `http://{COMPUTE_HOST}:{COMPUTE_PROXY_PORT}`, path-stripped. A real backend response (even 502) passes through untouched; a transport failure triggers a throttled wake and retries for up to 60s. |
| `/` (everything else) | Serves the UI's static build, `//go:embed`ded at build time. Unknown paths 404 (no SPA fallback). |

Requires the Gateway and compute host to share an L2 broadcast domain — WoL
doesn't route across subnets.

## Runtime environment

Read by the binary itself ([`internal/gateway/config.go`](../internal/gateway/config.go)),
which is the source of truth for these names and defaults. Bold variables are
required.

| Variable | Purpose |
|---|---|
| **`COMPUTE_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`COMPUTE_HOST`** | Host the `/server*` route forwards to. |
| **`COMPUTE_PROXY_PORT`** | Port on `COMPUTE_HOST` that `/server*` forwards to; no default. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `GATEWAY_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet ([`tailnet.AssertTailnetOnlyBind`](../internal/tailnet/bindsafety.go)). |
| `GATEWAY_PORT` | Listen port (default `5000`). |

## Deployment

[`deploy/deploy_gateway.py`](../deploy/deploy_gateway.py) builds the UI, embeds
it into the Gateway binary, cross-compiles for the Pi Zero W
(`GOARCH=arm GOARM=6` — ARM1176; a Pi Zero 2 W would need `GOARCH=arm64`), and
ships only the binary plus a systemd unit. The device never runs a Go or Node
toolchain. Because the binary does all three jobs, this is the only Deploy file
for the device.

It then runs `tailscale serve --bg --https=443 localhost:$GATEWAY_PORT`, guarded
by `common.TailscaleServeStatus` so a second run is a no-op. `tailscale serve`'s
CLI/JSON shape has moved across Tailscale versions — re-verify against the
installed version before trusting it on a new device.

### Deploy-time variables

Set on the dev machine, not the device. The binary's own defaults apply to
anything absent here — deploy never sets `GATEWAY_HOST`, so the loopback default
always wins. Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| `GATEWAY_PORT` | `5000` | Listen port; pinned explicitly (rather than relying on the binary's matching default) since `tailscale serve` has to point at the right port |
| **`COMPUTE_MAC_ADDRESS`** | — | WoL target MAC |
| `GATEWAY_PROXY_HOST` | `main-server.tailnet` | Host `/server*` forwards to. Deliberately *not* named `COMPUTE_HOST`: with no `env_prefix`, that name would alias `InventorySettings`' SSH target. Shipped to the device as the binary's `COMPUTE_HOST` |
| **`COMPUTE_PROXY_PORT`** | — | Port on `GATEWAY_PROXY_HOST` that `/server*` forwards to — the Compute API's own listen port. Required, not defaulted: no single correct target across every device the binary might run on |
| **`GRAFANA_CLOUD_LOKI_URL`** | — | Grafana Cloud's Loki push endpoint |
| **`GRAFANA_CLOUD_LOKI_USER`** | — | Loki basic-auth username (numeric instance ID) |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | — | Grafana Cloud Access Policy token, scoped to `logs:write` |

`VITE_COMPUTE_API_URL` (baked into the UI build) is derived from
`InventorySettings.compute_tailnet_host`, not a separate secret.

See [`deploy.md`](deploy.md) for how to run a Deploy and how it's tested.
