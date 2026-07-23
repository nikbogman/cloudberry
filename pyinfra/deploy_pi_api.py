"""Deploys the Pi API to `pi`, plus delivers the UI's static build.

Uses the shared helper to pull this repo to a specific ref and run the
Pi API as an enabled systemd service. Also builds the UI on the
dev machine (`npm ci && npm run build`) and syncs only its static `dist/`
to `pi` -- the Pi never runs a Node/npm toolchain. Targetable in isolation:

    pyinfra inventory.py deploy_pi_api.py --limit pi
    pyinfra inventory.py deploy_pi_api.py --limit pi --dry

Required dev-machine env vars (fail fast if missing, only when targeting
`pi`/its `test` stand-in): `SERVER_MAC_ADDRESS`, `ALLOY_PUSH_URL` (the
Pi API's own config) plus `SERVER_API_ORIGIN`, baked into the UI
build as `VITE_SERVER_API_URL`. The UI, not this Pi's Caddy, routes to the
Server API (ADR-0001 rules out a Pi-side relay).

Note on `--dry`: the UI build (`local.shell` below) always runs,
even under `--dry` -- pyinfra's dry-run guarantee only covers remote
operations, not a local build step. It never touches `pi`, but isn't a
true no-op.
"""

from pyinfra import local
from pyinfra.operations import files

from common import has_device_role
from api_deploy import git_systemd_service
from settings import DeploySourceSettings, PiApiSecrets, PiApiSettings

source = DeploySourceSettings()
api_settings = PiApiSettings()
CLONE_DEST = "/srv/homelab"
APP_DIR = f"{CLONE_DEST}/services/pi-api"

if has_device_role("pi"):
    secrets = PiApiSecrets()

    git_systemd_service(
        repo_url=source.homelab_repo_url,
        ref=source.deploy_ref,
        dest=CLONE_DEST,
        working_directory=APP_DIR,
        unit_name="pi-api.service",
        description="Pi API -- sends Wake-on-LAN to the main server",
        start_command=(
            f"uv run flask --app pi_api.wsgi run "
            f"--host {api_settings.pi_api_host} --port {api_settings.pi_api_port}"
        ),
        environment={
            "SERVER_MAC_ADDRESS": secrets.server_mac_address,
            "ALLOY_PUSH_URL": secrets.alloy_push_url,
            "PI_API_HOST": api_settings.pi_api_host,
        },
        setup_commands=["uv sync"],
    )

    # VITE_SERVER_API_URL bakes in the real origin so the browser calls the
    # Server API directly (ADR-0001). VITE_PI_API_URL is left unset --
    # same-origin works since the Pi API shares this Caddy site.
    local.shell(
        f"cd ../services/ui && npm ci && "
        f"VITE_SERVER_API_URL={secrets.server_api_origin} npm run build",
        print_output=True,
    )

    files.sync(
        name="Sync the UI's static build to pi",
        src="../services/ui/dist",
        dest="/srv/ui",
        delete=True,
    )
