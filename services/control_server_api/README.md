# Control server API

Flask app that runs on the main server, exposing a reachability health check and the Suspend action. See the [Control server API](../../CONTEXT.md) entry in CONTEXT.md.

Fronted by its own `tailscale serve` instance — a distinct origin from the [Control UI](../control_ui), hence the CORS allow-list ([ADR-0001](../../docs/adr/0001-direct-browser-to-api-no-relay.md)).

## Endpoints

Both require the `Tailscale-User-Login` identity header ([ADR-0004](../../docs/adr/0004-tailnet-membership-authorization.md)).

- `GET /health` — returns `{"reachable": true}` (200). The Control UI polls this to decide whether the server is [Reachable](../../CONTEXT.md) and which of Wake/Suspend to offer. The first time a process observes itself reachable, it logs a `reachability_changed` event to Alloy — see `ReachabilityTracker` in [`app.py`](src/control_server_api/app.py) for why this can only ever fire once per process (it can't witness going *un*reachable, since it's asleep while that's true).
- `POST /suspend` — runs the configured suspend command (`systemctl suspend` by default, see [`suspend.py`](src/control_server_api/suspend.py)), logs `suspend_requested`/`suspend_succeeded`/`suspend_failed` to Alloy, and returns `{"suspend": "succeeded"}` (200) or `{"suspend": "failed"}` (500).

Suspend-to-RAM is the only sleep state this app can trigger — there is no code path anywhere in it that can cause a full shutdown (ACPI S5), and the suspend command is intentionally not configurable via environment variable, so a misconfiguration can't reintroduce one ([ADR-0002](../../docs/adr/0002-suspend-only-no-shutdown.md)).

## Configuration

Read from the environment by [`wsgi.py`](src/control_server_api/wsgi.py):

| Variable | Required | Purpose |
|---|---|---|
| `CONTROL_UI_ORIGIN` | yes | Origin allowed by the CORS policy — the Control UI's origin. |
| `ALLOY_PUSH_URL` | yes | Grafana Alloy endpoint for event logging. |
| `CONTROL_SERVER_API_HOST` | no (default `127.0.0.1`) | Bind address. Asserted at startup to be loopback or a tailnet address — the process refuses to start if this would expose it off-tailnet. |

## Development

```sh
uv sync --extra dev
uv run pytest
uv run mypy src
```

Run locally:

```sh
CONTROL_UI_ORIGIN=http://localhost:5173 ALLOY_PUSH_URL=http://localhost:4318 uv run flask --app control_server_api.wsgi run
```

## Deployment

Runs as a systemd unit on the main server, not Docker — a deliberate exception to how workload services run elsewhere in the homelab, chosen so this app stays controllable even while Docker itself is being redeployed ([ADR-0007](../../docs/adr/0007-systemd-not-docker-for-control-apis.md)). Deployed by pulling this repo directly on-device and checking out a specific ref ([ADR-0006](../../docs/adr/0006-git-pull-deploy-with-offdevice-ui-build.md)). The pyinfra automation that will perform this deploy is spec'd (`.scratch/pyinfra-provisioning/spec.md`) but not yet built — deploying today is manual.
