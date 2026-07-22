"""Deploys the Control server API to `server`.

Uses the shared helper to pull this repo to a specific ref and run the
Control server API as an enabled systemd service. Structurally similar to
`deploy_control_pi_api.py` but a distinct application on a distinct Host
group. Targetable in isolation:

    pyinfra inventory.py deploy_control_server_api.py --limit server
    pyinfra inventory.py deploy_control_server_api.py --limit server --dry

Required dev-machine env vars (fail fast if missing, only when targeting
`server`/its `test` stand-in): `CONTROL_UI_ORIGIN`, `ALLOY_PUSH_URL` (the
Control server API's own config).
"""

from common import has_device_role
from control_api_deploy import git_systemd_service
from settings import ControlServerApiSecrets, ControlServerApiSettings, DeploySourceSettings

source = DeploySourceSettings()
api_settings = ControlServerApiSettings()
CLONE_DEST = "/srv/homelab"
APP_DIR = f"{CLONE_DEST}/services/control_server_api"

if has_device_role("server"):
    secrets = ControlServerApiSecrets()

    git_systemd_service(
        repo_url=source.homelab_repo_url,
        ref=source.deploy_ref,
        dest=CLONE_DEST,
        working_directory=APP_DIR,
        unit_name="control-server-api.service",
        description="Control server API -- exposes Suspend and the reachability health check",
        start_command=(
            f"uv run flask --app control_server_api.wsgi run "
            f"--host {api_settings.control_server_api_host} --port {api_settings.control_server_api_port}"
        ),
        environment={
            "CONTROL_UI_ORIGIN": secrets.control_ui_origin,
            "ALLOY_PUSH_URL": secrets.alloy_push_url,
            "CONTROL_SERVER_API_HOST": api_settings.control_server_api_host,
        },
        setup_commands=["uv sync"],
    )
