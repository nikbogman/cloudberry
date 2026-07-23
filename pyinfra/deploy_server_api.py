"""Deploys the Server API to `server`.

Uses the shared helper to pull this repo to a specific ref and run the
Server API as an enabled systemd service. Structurally similar to
`deploy_pi_api.py` but a distinct application on a distinct Host group.
Targetable in isolation:

    pyinfra inventory.py deploy_server_api.py --limit server
    pyinfra inventory.py deploy_server_api.py --limit server --dry

Required dev-machine env vars (fail fast if missing, only when targeting
`server`/its `test` stand-in): `UI_ORIGIN`, `ALLOY_PUSH_URL` (the
Server API's own config).
"""

from common import has_device_role
from api_deploy import git_systemd_service
from settings import DeploySourceSettings, ServerApiSecrets, ServerApiSettings

source = DeploySourceSettings()
api_settings = ServerApiSettings()
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
            "UI_ORIGIN": secrets.ui_origin,
            "ALLOY_PUSH_URL": secrets.alloy_push_url,
            "SERVER_API_HOST": api_settings.server_api_host,
        },
        setup_commands=["uv sync"],
    )
