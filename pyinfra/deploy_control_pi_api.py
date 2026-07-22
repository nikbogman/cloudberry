"""Deploys the Control Pi API to `pi`, plus delivers the Control UI's
static build.

Uses the shared helper to pull this repo to a specific ref and run the
Control Pi API as an enabled systemd service. Also builds the Control UI
on the dev machine (`npm ci && npm run build`) and syncs only its static
`dist/` to `pi` -- the Pi never runs a Node/npm toolchain. Targetable in
isolation:

    pyinfra inventory.py deploy_control_pi_api.py --limit pi
    pyinfra inventory.py deploy_control_pi_api.py --limit pi --dry

Required dev-machine env vars (fail fast if missing, only when targeting
`pi`/its `test` stand-in): `SERVER_MAC_ADDRESS`, `ALLOY_PUSH_URL` (the
Control Pi API's own config) plus `CONTROL_SERVER_API_ORIGIN`, baked into
the Control UI build as `VITE_CONTROL_SERVER_API_URL`. The UI, not this
Pi's Caddy, routes to the Control server API (ADR-0001 rules out a Pi-side
relay).

Note on `--dry`: the Control UI build (`local.shell` below) always runs,
even under `--dry` -- pyinfra's dry-run guarantee only covers remote
operations, not a local build step. It never touches `pi`, but isn't a
true no-op.
"""

from pyinfra import local
from pyinfra.operations import files

from common import has_device_role
from control_api_deploy import git_systemd_service
from settings import ControlPiApiSecrets, ControlPiApiSettings, DeploySourceSettings

source = DeploySourceSettings()
api_settings = ControlPiApiSettings()
CLONE_DEST = "/srv/homelab"
APP_DIR = f"{CLONE_DEST}/services/control_pi_api"

if has_device_role("pi"):
    secrets = ControlPiApiSecrets()

    git_systemd_service(
        repo_url=source.homelab_repo_url,
        ref=source.deploy_ref,
        dest=CLONE_DEST,
        working_directory=APP_DIR,
        unit_name="control-pi-api.service",
        description="Control Pi API -- sends Wake-on-LAN to the main server",
        start_command=(
            f"uv run flask --app control_pi_api.wsgi run "
            f"--host {api_settings.control_pi_api_host} --port {api_settings.control_pi_api_port}"
        ),
        environment={
            "SERVER_MAC_ADDRESS": secrets.server_mac_address,
            "ALLOY_PUSH_URL": secrets.alloy_push_url,
            "CONTROL_PI_API_HOST": api_settings.control_pi_api_host,
        },
        setup_commands=["uv sync"],
    )

    # VITE_CONTROL_SERVER_API_URL bakes in the real origin so the browser
    # calls the Control server API directly (ADR-0001). VITE_CONTROL_PI_API_URL
    # is left unset -- same-origin works since the Pi API shares this Caddy site.
    local.shell(
        f"cd ../services/control_ui && npm ci && "
        f"VITE_CONTROL_SERVER_API_URL={secrets.control_server_api_origin} npm run build",
        print_output=True,
    )

    files.sync(
        name="Sync the Control UI's static build to pi",
        src="../services/control_ui/dist",
        dest="/srv/control-ui",
        delete=True,
    )
