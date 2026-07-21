"""Ticket 05: deploys the Control Pi API to `pi`, plus delivers the
Control UI's static build.

Uses the shared helper (ticket 04) to pull this repo to a specific ref and
run the Control Pi API as an enabled systemd service. Also builds the
Control UI on the dev machine (`npm ci && npm run build`, ADR-0006) and
syncs only its static `dist/` output to `pi` -- the Pi never runs a
Node/npm toolchain. Targetable in isolation:

    pyinfra inventory.py deploy_control_pi_api.py --limit pi
    pyinfra inventory.py deploy_control_pi_api.py --limit pi --dry

Required dev-machine environment variables (fail fast if missing, only
when actually targeting `pi`/its `test` stand-in -- see `common.has_device_kind`):
`SERVER_MAC_ADDRESS`, `ALLOY_PUSH_URL` -- the Control Pi API's own required
config (`services/control_pi_api/README.md`) -- plus `CONTROL_SERVER_API_ORIGIN`,
baked into the Control UI build as `VITE_CONTROL_SERVER_API_URL`
(`services/control_ui/README.md`). The UI, not this Pi's Caddy, is what
"routes to" the Control server API (ADR-0001 rules out a Pi-side relay;
see `deploy_caddy.py`'s docstring).

Note on `--dry`: the Control UI build (`local.shell` below) runs on the
dev machine unconditionally, including under `--dry` -- pyinfra's dry-run
guarantee ("nothing mutated") only covers the *remote* operations it
tracks and diffs (`files.sync` et al.), not a plain local build step. This
never touches `pi`, so it's consistent with `--dry`'s "preview without
touching a real device" intent, just worth knowing it's not a no-op.
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

    # Control UI (ADR-0006): built on the dev machine, only dist/ shipped.
    # VITE_CONTROL_SERVER_API_URL is baked in at build time so the browser
    # calls the Control server API's real origin directly (ADR-0001);
    # VITE_CONTROL_PI_API_URL is left unset -- relative/same-origin is
    # correct since the Control Pi API shares this Caddy site.
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
