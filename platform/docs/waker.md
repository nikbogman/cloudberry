# Waker

The always-on device (a Pi Zero W) and the Waker API binary it runs. Its one
job: send Wake-on-LAN to Sleeper. [Edge](edge.md) forwards the UI's Wake to it
over the tailnet; it stays on the Pi because the magic packet is a LAN
broadcast.

Vocabulary (Waker, Wake, Identity header) is in [`CONTEXT.md`](../../CONTEXT.md).

Code: [`internal/waker/`](../internal/waker),
entrypoint [`cmd/waker-api/`](../cmd/waker-api).

## Routes

| Route | Behavior |
|---|---|
| `POST /wake` | Requires the identity header or a `127.0.0.1` caller. Sends WoL to `SLEEPER_MAC_ADDRESS`, logs `wake_requested`/`_succeeded`/`_failed`, returns `{"wake": ...}`. |

Requires the Waker and Sleeper to share an L2 broadcast domain — WoL
doesn't route across subnets.

## Runtime environment

Read by the binary itself ([`internal/waker/config.go`](../internal/waker/config.go)),
which is the source of truth for these names and defaults. Bold variables are
required.

| Variable | Purpose |
|---|---|
| **`SLEEPER_MAC_ADDRESS`** | MAC address the magic packet targets. Validated at startup. |
| **`GRAFANA_CLOUD_LOKI_URL`** | Grafana Cloud's Loki push endpoint. |
| **`GRAFANA_CLOUD_LOKI_USER`** | Grafana Cloud Loki basic-auth username (numeric instance ID). |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | Grafana Cloud Access Policy token, scoped to `logs:write`. |
| `WAKER_HOST` | Bind address (default `127.0.0.1`). Must be loopback or tailnet ([`tailnet.AssertTailnetOnlyBind`](../internal/tailnet/bindsafety.go)). |
| `WAKER_PORT` | Listen port (default `5000`). |

## Deployment

[`deploy/platform/waker.py`](../../deploy/platform/waker.py)
cross-compiles for the Pi Zero W (`GOARCH=arm GOARM=6` — ARM1176; a Pi Zero 2 W
would need `GOARCH=arm64`), and ships only the binary plus a systemd unit. The
device never runs a Go toolchain.

It then runs `tailscale serve --bg --https=443 localhost:$WAKER_PORT`, guarded
by `common.TailscaleServeStatus` so a second run is a no-op. `tailscale serve`'s
CLI/JSON shape has moved across Tailscale versions — re-verify against the
installed version before trusting it on a new device.

### Deploy-time variables

Set on the dev machine, not the device — in `platform/.env` (see
`.env.example`), which `deploy/deploy.sh` sources alongside the repo root
`.env`. The binary's own defaults apply to anything absent here — deploy never
sets `WAKER_HOST`, so the loopback default always wins. Bold variables are
required.

| Variable | Default | Purpose |
|---|---|---|
| `WAKER_PORT` | `5000` | Listen port; pinned explicitly (rather than relying on the binary's matching default) since `tailscale serve` has to point at the right port |
| **`SLEEPER_MAC_ADDRESS`** | — | WoL target MAC |
| **`GRAFANA_CLOUD_LOKI_URL`** | — | Grafana Cloud's Loki push endpoint |
| **`GRAFANA_CLOUD_LOKI_USER`** | — | Loki basic-auth username (numeric instance ID) |
| **`GRAFANA_CLOUD_LOKI_API_KEY`** | — | Grafana Cloud Access Policy token, scoped to `logs:write` |

See [`deploy.md`](../../docs/deploy.md) for how to run a Deploy and how
it's tested.
