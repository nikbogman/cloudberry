# Server API

Flask app that runs on the main server, exposing a reachability health check and the Suspend action. See the [Server API](../../CONTEXT.md) entry in CONTEXT.md.

Fronted by its own `tailscale serve` instance — a distinct origin from the [UI](../ui), hence the CORS allow-list ([ADR-0001](../../docs/adr/0001-direct-browser-to-api-no-relay.md)).

## Endpoints

Both require the `Tailscale-User-Login` identity header ([ADR-0004](../../docs/adr/0004-tailnet-membership-authorization.md)).

- `GET /health` — returns `{"reachable": true}` (200). The UI polls this to decide whether the server is [Reachable](../../CONTEXT.md) and which of Wake/Suspend to offer.
- `POST /suspend` — runs the configured suspend command (`systemctl suspend` by default, see [`suspend.py`](src/server_api/suspend.py)) and returns `{"suspend": "succeeded"}` (200) or `{"suspend": "failed"}` (500).

Suspend-to-RAM is the only sleep state this app can trigger — there is no code path anywhere in it that can cause a full shutdown (ACPI S5), and the suspend command is intentionally not configurable via environment variable, so a misconfiguration can't reintroduce one ([ADR-0002](../../docs/adr/0002-suspend-only-no-shutdown.md)).

## Configuration

Read from the environment by [`wsgi.py`](src/server_api/wsgi.py):

| Variable | Required | Purpose |
|---|---|---|
| `UI_ORIGIN` | yes | Origin allowed by the CORS policy — the UI's origin. |
| `SERVER_API_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address — the process refuses to start if this would expose it off-tailnet. |

## Development

```sh
uv sync --extra dev
uv run pytest
uv run mypy src
```

Run locally:

```sh
UI_ORIGIN=http://localhost:5173 uv run flask --app server_api.wsgi run
```

## Deployment

Runs as a systemd unit on the main server, not Docker — a deliberate exception to how workload services run elsewhere in the homelab, chosen so this app stays controllable even while Docker itself is being redeployed ([ADR-0007](../../docs/adr/0007-systemd-not-docker-for-control-apis.md)). Deployed by pulling this repo directly on-device and checking out a specific ref ([ADR-0006](../../docs/adr/0006-git-pull-deploy-with-offdevice-ui-build.md)). Automated by pyinfra's `deploy_server_api.py` (`../../pyinfra/deploy_server_api.py`, per `.scratch/pyinfra-provisioning/spec.md`) — see [`pyinfra/README.md`](../../pyinfra/README.md).
