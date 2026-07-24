"""Deploys the Server API to `server`.

Uses the shared helper to pull this repo to a specific ref and run the
Server API as an enabled systemd service, then exposes it on the tailnet
over HTTPS via `tailscale serve` (CONTEXT.md's Server API is "fronted by
its own `tailscale serve` instance" -- this was previously a flagged gap,
see README's Known gaps). Structurally similar to `deploy_pi_api.py` but a
distinct application on a distinct Host group. Targetable in isolation:

    pyinfra inventory.py deploy_server_api.py --limit server
    pyinfra inventory.py deploy_server_api.py --limit server --dry

Assumes `server` is already joined to the tailnet (`deploy_tailscale.py`,
which `deploy.py` always runs first) -- `tailscale serve` needs a running,
joined `tailscaled`, and this file doesn't re-check that when run standalone,
same as every other Deploy file's implicit tailnet-address assumption.

`UI_ORIGIN` (the Server API's CORS allow-list entry) is no longer a
separate required secret -- it's derived from
`InventorySettings.pi_tailnet_host` (the Pi's real Tailscale MagicDNS name,
distinct from `pi_host`, which is just pyinfra's SSH target), since that's
exactly the origin the Pi's own `tailscale serve` instance (deploy_caddy.py)
exposes the UI at.

Required dev-machine env vars (fail fast if missing, only when targeting
`server`/its `test` stand-in): the `GRAFANA_CLOUD_LOKI_*` trio (the Server
API's own config).
"""

from pyinfra import host
from pyinfra.operations import server

from common import TailscaleServeStatus, has_device_role
from api_deploy import git_systemd_service
from settings import DeploySourceSettings, InventorySettings, ServerApiSecrets, ServerApiSettings

source = DeploySourceSettings()
api_settings = ServerApiSettings()
inventory = InventorySettings()
CLONE_DEST = "/srv/homelab"
APP_DIR = f"{CLONE_DEST}/services/server-api"

if has_device_role("server"):
    secrets = ServerApiSecrets()

    git_systemd_service(
        repo_url=source.homelab_repo_url,
        ref=source.deploy_ref,
        dest=CLONE_DEST,
        working_directory=APP_DIR,
        unit_name="server-api.service",
        description="Server API -- exposes Suspend and the reachability health check",
        start_command=(
            f"uv run flask --app server_api.wsgi run "
            f"--host {api_settings.server_api_host} --port {api_settings.server_api_port}"
        ),
        environment={
            "UI_ORIGIN": f"https://{inventory.pi_tailnet_host}",
            "SERVER_API_HOST": api_settings.server_api_host,
            "GRAFANA_CLOUD_LOKI_URL": secrets.grafana_cloud_loki_url,
            "GRAFANA_CLOUD_LOKI_USER": secrets.grafana_cloud_loki_user,
            "GRAFANA_CLOUD_LOKI_API_KEY": secrets.grafana_cloud_loki_api_key,
        },
        setup_commands=["uv sync"],
    )

    serve_target = f"localhost:{api_settings.server_api_port}"
    already_served = serve_target in host.get_fact(TailscaleServeStatus)
    if not already_served:
        server.shell(
            name="Expose the Server API on the tailnet via tailscale serve",
            commands=[f"tailscale serve --bg --https=443 {serve_target}"],
        )
