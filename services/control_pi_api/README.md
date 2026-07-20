# Control Pi API

Flask app that runs on the Pi Zero and sends a Wake-on-LAN magic packet to the main server. The backend for the [Control UI](../control_ui)'s Wake button — see the [Control Pi API](../../CONTEXT.md) entry in CONTEXT.md.

Path-routed under the same `tailscale serve` app as the Control UI, so it's same-origin with it — no CORS needed, unlike the [Control server API](../control_server_api) ([ADR-0001](../../docs/adr/0001-direct-browser-to-api-no-relay.md)).

## Endpoint

`POST /wake` — requires the `Tailscale-User-Login` identity header ([ADR-0004](../../docs/adr/0004-tailnet-membership-authorization.md)). Sends a WoL magic packet to the configured `SERVER_MAC_ADDRESS`, logs a `wake_requested`/`wake_succeeded`/`wake_failed` event to Alloy, and returns `{"wake": "succeeded"}` (200) or `{"wake": "failed"}` (500).

This is one of two independent wake triggers in the system — the other is the [Auto-wake proxy](../auto_wake_proxy), which wakes the server automatically on any proxied request. Neither calls the other ([ADR-0005](../../docs/adr/0005-dual-wake-paths.md)).

Requires the Pi and the main server to share an L2 broadcast domain ([ADR-0003](../../docs/adr/0003-wol-same-broadcast-domain.md)) — WoL packets don't route across subnets.

## Configuration

Read from the environment by [`wsgi.py`](src/control_pi_api/wsgi.py):

| Variable | Required | Purpose |
|---|---|---|
| `SERVER_MAC_ADDRESS` | yes | MAC address the magic packet targets. Validated at startup — a malformed value fails fast, not mid-request. |
| `ALLOY_PUSH_URL` | yes | Grafana Alloy endpoint for event logging. |
| `CONTROL_PI_API_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address ([`assert_tailnet_only_bind`](../shared/src/control_plane_shared/bind_safety.py)) — the process refuses to start if this would expose it off-tailnet. |

## Development

```sh
uv sync --extra dev
uv run pytest
uv run mypy src
```

Run locally:

```sh
SERVER_MAC_ADDRESS=AA:BB:CC:DD:EE:FF ALLOY_PUSH_URL=http://localhost:4318 uv run flask --app control_pi_api.wsgi run
```

## Deployment

Runs as a systemd unit on the Pi Zero, not Docker — a deliberate exception to how workload services run elsewhere in the homelab ([ADR-0007](../../docs/adr/0007-systemd-not-docker-for-control-apis.md)). Deployed by pulling this repo directly on-device and checking out a specific ref ([ADR-0006](../../docs/adr/0006-git-pull-deploy-with-offdevice-ui-build.md)). The pyinfra automation that will perform this deploy is spec'd (`.scratch/pyinfra-provisioning/spec.md`) but not yet built — deploying today is manual.
