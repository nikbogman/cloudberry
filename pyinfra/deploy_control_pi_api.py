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

import os

from pyinfra import local
from pyinfra.operations import files

from common import has_device_kind
from control_api_deploy import git_systemd_service

REPO_URL = os.environ.get("HOMELAB_REPO_URL", "git@github.com:nikbogman/homelab.git")
REF = os.environ.get("DEPLOY_REF", "main")
CLONE_DEST = "/srv/homelab"
APP_DIR = f"{CLONE_DEST}/services/control_pi_api"
BIND_HOST = os.environ.get("CONTROL_PI_API_HOST", "127.0.0.1")
BIND_PORT = os.environ.get("CONTROL_PI_API_PORT", "5000")

if has_device_kind("pi"):
    git_systemd_service(
        repo_url=REPO_URL,
        ref=REF,
        dest=CLONE_DEST,
        working_directory=APP_DIR,
        unit_name="control-pi-api.service",
        description="Control Pi API -- sends Wake-on-LAN to the main server",
        start_command=f"uv run flask --app control_pi_api.wsgi run --host {BIND_HOST} --port {BIND_PORT}",
        environment={
            "SERVER_MAC_ADDRESS": os.environ["SERVER_MAC_ADDRESS"],
            "ALLOY_PUSH_URL": os.environ["ALLOY_PUSH_URL"],
            "CONTROL_PI_API_HOST": BIND_HOST,
        },
        setup_commands=["uv sync"],
    )

    # VITE_CONTROL_SERVER_API_URL bakes in the real origin so the browser
    # calls the Control server API directly (ADR-0001). VITE_CONTROL_PI_API_URL
    # is left unset -- same-origin works since the Pi API shares this Caddy site.
    control_server_api_origin = os.environ["CONTROL_SERVER_API_ORIGIN"]
    local.shell(
        f"cd ../services/control_ui && npm ci && VITE_CONTROL_SERVER_API_URL={control_server_api_origin} npm run build",
        print_output=True,
    )

    files.sync(
        name="Sync the Control UI's static build to pi",
        src="../services/control_ui/dist",
        dest="/srv/control-ui",
        delete=True,
    )
