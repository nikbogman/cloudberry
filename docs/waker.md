# Waker

The always-on device (a Pi Zero W) and the Waker API binary it runs. One
process does two jobs: serves the UI's static files and sends Wake-on-LAN to
Compute. Same-origin with the [UI](ui.md), so no CORS entry is needed.

It is deliberately *not* in the workload traffic path — the Pi is on Wi-Fi,
so proxying workload traffic through it costs a hop and a bottleneck for
nothing. Browsers reach workload containers on Compute directly on the tailnet.

Vocabulary (Waker, Wake, Identity header) is in [`CONTEXT.md`](../CONTEXT.md).

Code: [`internal/waker/`](../internal/waker),
entrypoint [`cmd/waker-api/`](../cmd/waker-api).

## Routes

| Route | Behavior |
|---|---|
| `POST /wake` | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `COMPUTE_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |
| `GET /config.js` | One line of JS setting `window.COMPUTE_API_URL` from the env var, so the UI learns Compute's origin without anything being generated into `ui/`. |
| `/` (everything else) | Serves the UI's static files, `//go:embed`ded from `ui/` at build time. Unknown paths 404 (no SPA fallback). |

Requires the Waker and compute host to share an L2 broadcast domain — WoL
doesn't route across subnets.

## Runtime environment

Read by the binary itself ([`internal/waker/config.go`](../internal/waker/config.go)),
which is the source of truth for these names and defaults. Bold variables are
required.

| Variable | Purpose |
|---|---|
| **`COMPUTE_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `WAKER_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet ([`tailnet.AssertTailnetOnlyBind`](../internal/tailnet/bindsafety.go)). |
| `WAKER_PORT` | Listen port (default `5000`). |

## Deployment

[`deploy/deploy_waker.py`](../deploy/deploy_waker.py) builds the UI, embeds
it into the Waker binary, cross-compiles for the Pi Zero W
(`GOARCH=arm GOARM=6` — ARM1176; a Pi Zero 2 W would need `GOARCH=arm64`), and
ships only the binary plus a systemd unit. The device never runs a Go or Node
toolchain. Because the binary does both jobs, this is the only Deploy file for
the device.

It then runs `tailscale serve --bg --https=443 localhost:$WAKER_PORT`, guarded
by `common.TailscaleServeStatus` so a second run is a no-op. `tailscale serve`'s
CLI/JSON shape has moved across Tailscale versions — re-verify against the
installed version before trusting it on a new device.

### Deploy-time variables

Set on the dev machine, not the device. The binary's own defaults apply to
anything absent here — deploy never sets `WAKER_HOST`, so the loopback default
always wins. Bold variables are required.

| Variable | Default | Purpose |
|---|---|---|
| `WAKER_PORT` | `5000` | Listen port; pinned explicitly (rather than relying on the binary's matching default) since `tailscale serve` has to point at the right port |
| **`COMPUTE_MAC_ADDRESS`** | — | WoL target MAC |
| `COMPUTE_API_URL` | `''` | Compute API origin the UI calls; served to the browser at `/config.js`. Empty means same-origin, which only suits a local dev run |
| **`GRAFANA_CLOUD_LOKI_URL`** | — | Grafana Cloud's Loki push endpoint |
| **`GRAFANA_CLOUD_LOKI_USER`** | — | Loki basic-auth username (numeric instance ID) |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | — | Grafana Cloud Access Policy token, scoped to `logs:write` |

`COMPUTE_API_URL` is derived from `InventorySettings.compute_tailnet_host`, not
a separate secret.

See [`deploy.md`](deploy.md) for how to run a Deploy and how it's tested.
